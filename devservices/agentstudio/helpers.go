package agentstudio

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/microbus-io/bespa/widget"
	"github.com/microbus-io/dwarf/workflow"
	"github.com/microbus-io/errors"
	"github.com/microbus-io/fabric/coreservices/foreman/foremanapi"

	yaml "go.yaml.in/yaml/v3"
)

// statusChip renders a status string in a status-appropriate color.
func statusChip(status string) widget.Widget {
	switch status {
	case workflow.StatusCompleted, workflow.StatusRunning:
		return wf.TextStyle(status).WithColorOnPrimary()
	case workflow.StatusFailed, workflow.StatusCancelled:
		return wf.TextStyle(status).WithColorOnError()
	case workflow.StatusInterrupted:
		return wf.TextStyle(status).WithColorOnTertiary()
	case workflow.StatusPending:
		return wf.TextStyle(status).WithColorInverse()
	}
	return wf.Text(status)
}

// isLiveStatus reports whether a flow status indicates the flow may still
// produce updates worth re-rendering. interrupted is included because a
// Resume can transition it back to running while the page is open.
func isLiveStatus(status string) bool {
	switch status {
	case workflow.StatusCreated, workflow.StatusPending, workflow.StatusRunning, workflow.StatusInterrupted:
		return true
	}
	return false
}

// isForkableStatus reports whether a terminal status accepts foreman.Fork.
// interrupted is excluded: it is not terminal and is recovered via Resume, not Fork.
func isForkableStatus(status string) bool {
	switch status {
	case workflow.StatusCompleted, workflow.StatusFailed, workflow.StatusCancelled:
		return true
	}
	return false
}

func statusOf(o *workflow.FlowOutcome) string {
	if o == nil {
		return ""
	}
	return o.Status
}

func errorOf(o *workflow.FlowOutcome) string {
	if o == nil {
		return ""
	}
	return o.Error
}

func cancelReasonOf(o *workflow.FlowOutcome) string {
	if o == nil {
		return ""
	}
	return o.CancelReason
}

// formatDelta returns the elapsed time from origin to t, as a short signed
// string like "+56ms", "+2.1s", "+1m4s", or "0" for the origin row.
func formatDelta(t, origin time.Time) string {
	if t.IsZero() || origin.IsZero() {
		return ""
	}
	d := t.Sub(origin)
	if d == 0 {
		return "0"
	}
	sign := "+"
	if d < 0 {
		sign = "-"
		d = -d
	}
	switch {
	case d < time.Millisecond:
		return fmt.Sprintf("%s%dus", sign, d/time.Microsecond)
	case d < time.Second:
		return fmt.Sprintf("%s%dms", sign, d/time.Millisecond)
	case d < time.Minute:
		return fmt.Sprintf("%s%.1fs", sign, d.Seconds())
	case d < time.Hour:
		m := int(d / time.Minute)
		s := int((d % time.Minute) / time.Second)
		return fmt.Sprintf("%s%dm%ds", sign, m, s)
	default:
		h := int(d / time.Hour)
		m := int((d % time.Hour) / time.Minute)
		return fmt.Sprintf("%s%dh%dm", sign, h, m)
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-3] + "..."
}

// expandable returns the widgets that render value with a "Show more" /
// "Show less" toggle when it exceeds expandableThreshold.
func expandable(r *http.Request, stateKey, value string, style func(s string) any) []any {
	if len(value) <= 384 {
		return []any{style(value)}
	}
	collapsed := wf.StateOf(r).Get(stateKey) == ""
	short := value[:256] + "..."
	shortChildren := []any{""}
	fullChildren := []any{""}
	if collapsed {
		shortChildren = []any{style(short), " "}
	} else {
		fullChildren = []any{style(value), " "}
	}
	return []any{
		wf.Collection(shortChildren...).HideIfNotEq(r, stateKey, "").RedrawIfChanged(r, stateKey),
		wf.Link("?"+stateKey+"=1").Add(wf.HTMLUnsafe("More &raquo;"), wf.Collection(wf.HTMLUnsafe("&nbsp;"), "(", (len(value)+512)/1024, " KB)").HideIf(len(value) < 1024)).HideIfNotEq(r, stateKey, "").RedrawIfChanged(r, stateKey),
		wf.Collection(fullChildren...).HideIfEq(r, stateKey, "").RedrawIfChanged(r, stateKey),
		wf.Link("?"+stateKey+"=").Add(wf.HTMLUnsafe("&laquo; Less")).HideIfEq(r, stateKey, "").RedrawIfChanged(r, stateKey),
	}
}

// stateKeyFor builds a URL-safe state variable name from a prefix and a
// JSON state field name.
func stateKeyFor(prefix, k string) string {
	var b strings.Builder
	b.WriteString(prefix)
	b.WriteByte('_')
	for _, c := range k {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') {
			b.WriteRune(c)
		} else {
			b.WriteByte('_')
		}
	}
	return b.String()
}

// stateField builds a form row for one state key. The left side shows the key
// name and a CopyToClipboard icon that copies "name: value" verbatim; the right
// side shows the value through the expandable "Show more" toggle.
func stateField(r *http.Request, stateKey, name, value string, style func(s string) any) any {
	return wf.Field().
		AddLeft(name, " ", wf.CopyToClipboard(name+": "+value)).
		AddRight(expandable(r, stateKey, value, style)...)
}

// stateMap decodes a flow state into a plain map. The studio renders state generically, key by key,
// so it works off the decoded map rather than the typed State accessors.
func stateMap(s workflow.State) map[string]any {
	m := map[string]any{}
	_ = s.Parse(&m)
	return m
}

// flowDuration is the wall-clock time a listed flow has been running, from StartedAt to UpdatedAt.
func flowDuration(f foremanapi.FlowSummary) time.Duration {
	if f.StartedAt.IsZero() || f.UpdatedAt.IsZero() {
		return 0
	}
	d := f.UpdatedAt.Sub(f.StartedAt)
	if d < 0 {
		return 0
	}
	return d
}

// renderStateForm builds a Form with one field per key in m (sorted).
func renderStateForm(r *http.Request, prefix string, m map[string]any, dimKeys map[string]bool) any {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	form := wf.Form()
	for _, k := range keys {
		style := func(s string) any { return wf.Text(s) }
		if dimKeys[k] {
			style = func(s string) any { return wf.TextStyle(s).WithColorDeemphasized() }
		}
		form.Add(stateField(r, stateKeyFor(prefix, k), k, jsonValueString(m[k], true), style))
	}
	return form
}

// mergedSortedKeys returns the union of the keys of two maps, sorted.
func mergedSortedKeys(a, b map[string]any) []string {
	seen := make(map[string]struct{}, len(a)+len(b))
	for k := range a {
		seen[k] = struct{}{}
	}
	for k := range b {
		seen[k] = struct{}{}
	}
	out := make([]string, 0, len(seen))
	for k := range seen {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// jsonValueString renders an arbitrary JSON-decoded value as a display string.
func jsonValueString(v any, present bool) string {
	if !present {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("%v", v)
	}
	return string(b)
}

type flatStep struct {
	step  foremanapi.FlowStep
	depth int // subgraph nesting depth; 0 for root-flow steps
}

// flattenSteps walks SubHistory and returns leaf steps with their nesting depth.
// Subgraph wrapper steps are dropped; their child steps carry the meaningful timing.
func flattenSteps(steps []foremanapi.FlowStep, depth int) []flatStep {
	out := make([]flatStep, 0, len(steps))
	for _, s := range steps {
		if s.Subgraph && len(s.SubHistory) > 0 {
			out = append(out, flattenSteps(s.SubHistory, depth+1)...)
			continue
		}
		out = append(out, flatStep{step: s, depth: depth})
	}
	return out
}

// countSteps recursively counts steps including SubHistory.
func countSteps(steps []foremanapi.FlowStep) int {
	n := 0
	for _, s := range steps {
		n++
		if s.Subgraph && len(s.SubHistory) > 0 {
			n += countSteps(s.SubHistory)
		}
	}
	return n
}

// parseStateRaw parses a textarea payload as JSON first and falls back to
// YAML on failure. Empty input returns (nil, nil) so an empty form means
// "no initial state".
func parseStateRaw(raw string) (any, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	var v any
	jsonErr := json.Unmarshal([]byte(raw), &v)
	if jsonErr == nil {
		return v, nil
	}
	yamlErr := yaml.Unmarshal([]byte(raw), &v)
	if yamlErr == nil {
		return v, nil
	}
	return nil, errors.New("state is neither valid JSON (%s) nor YAML (%s)", jsonErr.Error(), yamlErr.Error())
}

// threadIsContinuable reports whether the latest flow in flowKey's thread is
// completed and therefore a valid foreman.Continue target. Failures are
// swallowed (the button just renders disabled) since this is best-effort UI gating.
func (svc *Service) threadIsContinuable(r *http.Request, flowKey string) bool {
	flows, _, err := svc.foreman.List(r.Context(), foremanapi.Query{
		ThreadKey: flowKey,
		Limit:     1,
	})
	if err != nil || len(flows) == 0 {
		return false
	}
	return flows[0].Status == workflow.StatusCompleted
}
