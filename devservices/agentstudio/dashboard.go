package agentstudio

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/microbus-io/bespa/chart"

	"github.com/microbus-io/dwarf/workflow"
	"github.com/microbus-io/fabric/coreservices/foreman/foremanapi"
)

// Dashboard tuning.
const (
	dashboardHistoryLimit = 200 // cap History fan-out per page render
	dashboardBuckets      = 48  // time buckets per chart across the window
	dashboardTopTasks     = 20  // heatmap row cap, top by step count
	dashboardTopErrorRows = 15  // error-timeline y-axis cap, top by event count
	dashboardChartHeight  = "360px"
)

// Status palette. Hardcoded brand-teal-derived hexes that read well on both
// light and dark backgrounds (echarts canvas does not resolve CSS vars).
var statusPalette = map[string]string{
	workflow.StatusCompleted:   "#32a7c1",
	workflow.StatusRunning:     "#7cc9d9",
	workflow.StatusPending:     "#cae6ec",
	workflow.StatusInterrupted: "#f0ad4e",
	workflow.StatusFailed:      "#d9534f",
	workflow.StatusTerminated:  "#8d6e63",
	workflow.StatusCancelled:   "#9e9e9e",
}

// Order in which statuses stack and appear in the legend.
var statusOrder = []string{
	workflow.StatusCompleted,
	workflow.StatusRunning,
	workflow.StatusPending,
	workflow.StatusInterrupted,
	workflow.StatusFailed,
	workflow.StatusTerminated,
	workflow.StatusCancelled,
}

func dashboardWindow(key string) time.Duration {
	if key == "24h" {
		return 24 * time.Hour
	}
	return 8 * time.Hour
}

func windowKey(window time.Duration) string {
	if window >= 24*time.Hour {
		return "24h"
	}
	return "8h"
}

// flattenAllSteps walks a flow's History (including subgraph SubHistory) and
// appends every leaf step. Subgraph wrapper steps are skipped; their children
// carry the meaningful timing.
func flattenAllSteps(steps []foremanapi.FlowStep, out []foremanapi.FlowStep) []foremanapi.FlowStep {
	for _, s := range steps {
		if s.Subgraph && len(s.SubHistory) > 0 {
			out = flattenAllSteps(s.SubHistory, out)
			continue
		}
		out = append(out, s)
	}
	return out
}

// bucketize returns the index of t in a window of N buckets ending at end.
// Returns -1 when t falls outside the window.
func bucketize(t, end time.Time, window time.Duration, buckets int) int {
	if t.IsZero() {
		return -1
	}
	start := end.Add(-window)
	if t.Before(start) || !t.Before(end) {
		// Clamp the right edge inclusively so a step that just ended at end falls in the last bucket.
		if t.Equal(end) {
			return buckets - 1
		}
		return -1
	}
	delta := t.Sub(start)
	idx := int(delta * time.Duration(buckets) / window)
	if idx < 0 {
		idx = 0
	}
	if idx >= buckets {
		idx = buckets - 1
	}
	return idx
}

// bucketLabels returns the right-edge time of each bucket, pre-formatted for
// display on an echarts category axis. The format includes the date only when
// the window exceeds a calendar day, otherwise just "HH:MM" in the local
// timezone of the server (which is fine for a dev console).
func bucketLabels(end time.Time, window time.Duration, buckets int) []string {
	step := window / time.Duration(buckets)
	out := make([]string, buckets)
	start := end.Add(-window)
	layout := "15:04"
	if window > 24*time.Hour {
		layout = "01/02 15:04"
	}
	for i := 0; i < buckets; i++ {
		out[i] = start.Add(step * time.Duration(i+1)).Local().Format(layout)
	}
	return out
}

// buildStatusTimelineChart renders flow counts bucketed by CreatedAt, stacked by status.
func buildStatusTimelineChart(flows []foremanapi.FlowSummary, window time.Duration) *chart.ChartWidget {
	end := time.Now().UTC()
	xs := bucketLabels(end, window, dashboardBuckets)

	counts := make(map[string][]int, len(statusOrder))
	for _, st := range statusOrder {
		counts[st] = make([]int, dashboardBuckets)
	}
	for _, f := range flows {
		idx := bucketize(f.CreatedAt, end, window, dashboardBuckets)
		if idx < 0 {
			continue
		}
		s, ok := counts[f.Status]
		if !ok {
			continue
		}
		s[idx]++
	}

	xsJSON, _ := json.Marshal(xs)
	var seriesJS []string
	for _, st := range statusOrder {
		c := counts[st]
		if sliceSum(c) == 0 {
			continue
		}
		dataJSON, _ := json.Marshal(c)
		seriesJS = append(seriesJS, fmt.Sprintf(
			`{name:'%s',type:'bar',stack:'flows',itemStyle:{color:'%s'},data:%s}`,
			st, statusPalette[st], dataJSON,
		))
	}
	seriesBlock := strings.Join(seriesJS, ",")

	cfg := fmt.Sprintf(`{
		title: {text: 'Flows started — by status', left: 10, top: 6, textStyle: {fontSize: 13, fontWeight: 'normal'}},
		tooltip: {trigger: 'axis', axisPointer: {type: 'shadow'}},
		legend: {bottom: 0, type: 'scroll'},
		grid: {top: 40, bottom: 40, left: 50, right: 16, containLabel: true},
		xAxis: {type: 'category', data: %s, axisLabel: {hideOverlap: true}},
		yAxis: {type: 'value', minInterval: 1},
		series: [%s]
	}`, xsJSON, seriesBlock)

	return wf.Chart(cfg, nil).WithHeight(dashboardChartHeight)
}

// buildErrorTimelineChart renders failed/terminated/cancelled flow events as a scatter,
// y-axis = task name, x-axis = UpdatedAt.
func buildErrorTimelineChart(flows []foremanapi.FlowSummary, window time.Duration) *chart.ChartWidget {
	end := time.Now().UTC()
	start := end.Add(-window)

	type ev struct {
		x      int64
		y      string
		status string
		msg    string
	}
	var events []ev
	taskCounts := map[string]int{}
	for _, f := range flows {
		if f.Status != workflow.StatusFailed && f.Status != workflow.StatusTerminated && f.Status != workflow.StatusCancelled {
			continue
		}
		if f.UpdatedAt.Before(start) || f.UpdatedAt.After(end) {
			continue
		}
		task := f.StoppedTaskName
		if task == "" {
			task = "(no task)"
		}
		msg := f.Error
		if msg == "" {
			msg = f.TerminateReason
		}
		if msg == "" {
			msg = f.CancelReason
		}
		events = append(events, ev{
			x: f.UpdatedAt.UnixMilli(), y: task, status: f.Status, msg: msg,
		})
		taskCounts[task]++
	}

	taskOrder := topKeysByCount(taskCounts, dashboardTopErrorRows)
	taskSet := make(map[string]bool, len(taskOrder))
	for _, t := range taskOrder {
		taskSet[t] = true
	}

	type pt struct {
		Name  string `json:"name"`
		Value [2]any `json:"value"`
	}
	failed := []pt{}
	terminated := []pt{}
	cancelled := []pt{}
	for _, e := range events {
		if !taskSet[e.y] {
			continue
		}
		clock := time.UnixMilli(e.x).Local().Format("15:04:05")
		msg := e.msg
		if msg == "" {
			msg = "(no message)"
		}
		p := pt{
			Name:  fmt.Sprintf("%s — %s — %s", clock, e.y, msg),
			Value: [2]any{e.x, e.y},
		}
		switch e.status {
		case workflow.StatusFailed:
			failed = append(failed, p)
		case workflow.StatusTerminated:
			terminated = append(terminated, p)
		default:
			cancelled = append(cancelled, p)
		}
	}

	yJSON, _ := json.Marshal(taskOrder)
	failedJSON, _ := json.Marshal(failed)
	terminatedJSON, _ := json.Marshal(terminated)
	cancelledJSON, _ := json.Marshal(cancelled)
	startMs := start.UnixMilli()
	endMs := end.UnixMilli()

	cfg := fmt.Sprintf(`{
		title: {text: 'Error events — failed / terminated / cancelled', left: 10, top: 6, textStyle: {fontSize: 13, fontWeight: 'normal'}},
		tooltip: {trigger: 'item', formatter: '{b}'},
		legend: {bottom: 0},
		grid: {top: 40, bottom: 40, left: 50, right: 16, containLabel: true},
		xAxis: {type: 'time', min: %d, max: %d},
		yAxis: {type: 'category', data: %s, axisLabel: {fontSize: 11}},
		series: [
			{name: 'failed', type: 'scatter', symbolSize: 10, itemStyle: {color: '%s'}, data: %s},
			{name: 'terminated', type: 'scatter', symbolSize: 10, itemStyle: {color: '%s'}, data: %s},
			{name: 'cancelled', type: 'scatter', symbolSize: 10, itemStyle: {color: '%s'}, data: %s}
		]
	}`, startMs, endMs, yJSON, statusPalette[workflow.StatusFailed], failedJSON,
		statusPalette[workflow.StatusTerminated], terminatedJSON,
		statusPalette[workflow.StatusCancelled], cancelledJSON)

	return wf.Chart(cfg, nil).WithHeight(dashboardChartHeight)
}

// buildTaskHeatmapChart renders step counts by (task, time bucket).
func buildTaskHeatmapChart(histories [][]foremanapi.FlowStep, window time.Duration) *chart.ChartWidget {
	end := time.Now().UTC()
	xs := bucketLabels(end, window, dashboardBuckets)

	taskCounts := map[string]int{}
	var flat []foremanapi.FlowStep
	for _, h := range histories {
		flat = flattenAllSteps(h, flat)
	}
	for _, s := range flat {
		taskCounts[s.TaskName]++
	}
	tasks := topKeysByCount(taskCounts, dashboardTopTasks)
	taskIdx := make(map[string]int, len(tasks))
	for i, t := range tasks {
		taskIdx[t] = i
	}

	cell := make(map[[2]int]int)
	maxV := 0
	for _, s := range flat {
		ti, ok := taskIdx[s.TaskName]
		if !ok {
			continue
		}
		bi := bucketize(s.CreatedAt, end, window, dashboardBuckets)
		if bi < 0 {
			continue
		}
		key := [2]int{bi, ti}
		cell[key]++
		if cell[key] > maxV {
			maxV = cell[key]
		}
	}

	type heatPt [3]int
	data := make([]heatPt, 0, len(cell))
	for k, v := range cell {
		data = append(data, heatPt{k[0], k[1], v})
	}

	xsJSON, _ := json.Marshal(xs)
	tasksJSON, _ := json.Marshal(tasks)
	dataJSON, _ := json.Marshal(data)
	if maxV < 1 {
		maxV = 1
	}

	cfg := fmt.Sprintf(`{
		title: {text: 'Task activity — steps per bucket', left: 10, top: 6, textStyle: {fontSize: 13, fontWeight: 'normal'}},
		tooltip: {position: 'top'},
		grid: {top: 40, bottom: 60, left: 50, right: 16, containLabel: true},
		xAxis: {type: 'category', data: %s, axisLabel: {hideOverlap: true}},
		yAxis: {type: 'category', data: %s, axisLabel: {fontSize: 11}},
		visualMap: {min: 0, max: %d, calculable: true, orient: 'horizontal', left: 'center', bottom: 0, inRange: {color: ['#eef6f8', '%s']}},
		series: [{type: 'heatmap', data: %s, emphasis: {itemStyle: {shadowBlur: 6, shadowColor: 'rgba(0,0,0,0.3)'}}}]
	}`, xsJSON, tasksJSON, maxV, statusPalette[workflow.StatusCompleted], dataJSON)

	height := dashboardChartHeight
	if rows := len(tasks); rows > 12 {
		height = fmt.Sprintf("%dpx", 40+24*rows+80)
	}
	return wf.Chart(cfg, nil).WithHeight(height)
}

// buildTaskBoxplotChart renders per-task step-duration distributions in ms.
func buildTaskBoxplotChart(histories [][]foremanapi.FlowStep) *chart.ChartWidget {
	var flat []foremanapi.FlowStep
	for _, h := range histories {
		flat = flattenAllSteps(h, flat)
	}

	durations := map[string][]float64{}
	for _, s := range flat {
		if s.Status != workflow.StatusCompleted {
			continue
		}
		if s.CreatedAt.IsZero() || s.UpdatedAt.IsZero() {
			continue
		}
		d := s.UpdatedAt.Sub(s.CreatedAt)
		if d <= 0 {
			continue
		}
		durations[s.TaskName] = append(durations[s.TaskName], float64(d.Milliseconds()))
	}

	// Sort tasks by median descending so slow tasks land on the left.
	type tStat struct {
		name string
		five [5]float64
		med  float64
	}
	stats := make([]tStat, 0, len(durations))
	for name, vs := range durations {
		if len(vs) < 1 {
			continue
		}
		sort.Float64s(vs)
		ts := tStat{name: name, five: fiveNumber(vs)}
		ts.med = ts.five[2]
		stats = append(stats, ts)
	}
	sort.Slice(stats, func(i, j int) bool { return stats[i].med > stats[j].med })
	if len(stats) > dashboardTopTasks {
		stats = stats[:dashboardTopTasks]
	}

	names := make([]string, len(stats))
	data := make([][5]float64, len(stats))
	for i, s := range stats {
		names[i] = s.name
		data[i] = s.five
	}
	namesJSON, _ := json.Marshal(names)
	dataJSON, _ := json.Marshal(data)

	cfg := fmt.Sprintf(`{
		title: {text: 'Task duration — completed steps (ms)', left: 10, top: 6, textStyle: {fontSize: 13, fontWeight: 'normal'}},
		tooltip: {trigger: 'item'},
		grid: {top: 40, bottom: 80, left: 60, right: 16, containLabel: true},
		xAxis: {type: 'category', data: %s, axisLabel: {rotate: 35, fontSize: 11, interval: 0}},
		yAxis: {type: 'log', name: 'ms', nameLocation: 'middle', nameGap: 40, min: 1},
		series: [{type: 'boxplot', itemStyle: {color: '%s', borderColor: '%s'}, data: %s}]
	}`, namesJSON, statusPalette[workflow.StatusCompleted], statusPalette[workflow.StatusCompleted], dataJSON)

	return wf.Chart(cfg, nil).WithHeight(dashboardChartHeight)
}

// fiveNumber returns [min, Q1, median, Q3, max] for a sorted slice.
func fiveNumber(sorted []float64) [5]float64 {
	n := len(sorted)
	if n == 0 {
		return [5]float64{}
	}
	q := func(p float64) float64 {
		if n == 1 {
			return sorted[0]
		}
		pos := p * float64(n-1)
		lo := int(pos)
		hi := lo + 1
		if hi >= n {
			return sorted[n-1]
		}
		frac := pos - float64(lo)
		return sorted[lo] + frac*(sorted[hi]-sorted[lo])
	}
	return [5]float64{sorted[0], q(0.25), q(0.5), q(0.75), sorted[n-1]}
}

func topKeysByCount(m map[string]int, limit int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if m[keys[i]] != m[keys[j]] {
			return m[keys[i]] > m[keys[j]]
		}
		return keys[i] < keys[j]
	})
	if len(keys) > limit {
		keys = keys[:limit]
	}
	// Reverse so highest counts land at the top of an echarts category axis (which draws bottom-up).
	for i, j := 0, len(keys)-1; i < j; i, j = i+1, j-1 {
		keys[i], keys[j] = keys[j], keys[i]
	}
	return keys
}

func sliceSum(xs []int) int {
	s := 0
	for _, v := range xs {
		s += v
	}
	return s
}
