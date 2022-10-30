package spec

import (
	"strings"
	"time"

	"github.com/microbus-io/fabric/errors"
)

// Handler is the spec of a callback handler.
// Web requests, lifecycle events, config changes, tickers, etc.
type Handler struct {
	Type   string `yaml:"-"`
	Exists bool   `yaml:"-"`

	// Web
	Signature   *Signature `yaml:"signature"`
	Description string     `yaml:"description"`
	Path        string     `yaml:"path"`
	Queue       string     `yaml:"queue"`

	// Config
	Default    string `yaml:"default"`
	Validation string `yaml:"validation"`
	Callback   bool   `yaml:"callback"`
	Secret     bool   `yaml:"secret"`

	// Ticker
	Interval   time.Duration `yaml:"interval"`
	TimeBudget time.Duration `yaml:"timeBudget"`
}

// UnmarshalYAML parses the handler.
func (cb *Handler) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type doppelganger Handler
	var dg doppelganger
	err := unmarshal(&dg)
	if err != nil {
		return errors.Trace(err)
	}
	*cb = Handler(dg)
	if cb.Path == "" {
		cb.Path = "/" + strings.ToLower(cb.Name())
	}
	if cb.Path == "^" {
		cb.Path = ""
	}
	cb.Path = strings.Replace(cb.Path, "...", strings.ToLower(cb.Name()), 1)
	return nil
}

// Name of the handler function.
func (cb *Handler) Name() string {
	return cb.Signature.Name
}

// In returns the input argument list as a string.
func (cb *Handler) In() string {
	var b strings.Builder
	for i, arg := range cb.Signature.InputArgs {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(arg.Name)
		b.WriteString(" ")
		b.WriteString(arg.Type)
	}
	return b.String()
}

// In returns the output argument list as a string.
func (cb *Handler) Out() string {
	var b strings.Builder
	for i, arg := range cb.Signature.OutputArgs {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(arg.Name)
		b.WriteString(" ")
		b.WriteString(arg.Type)
	}
	return b.String()
}

// Validate indicates if the specs are valid.
func (cb *Handler) Validate() error {
	err := cb.Signature.Validate()
	if err != nil {
		return errors.Trace(err)
	}

	switch cb.Type {
	case "web":
		if len(cb.Signature.InputArgs) != 0 || len(cb.Signature.OutputArgs) != 0 {
			return errors.Newf("invalid signature '%s'", cb.Signature)
		}
	case "config":
		if len(cb.Signature.InputArgs) != 0 || len(cb.Signature.OutputArgs) != 1 {
			return errors.Newf("invalid signature '%s'", cb.Signature)
		}
	case "ticker":
		if len(cb.Signature.InputArgs) != 0 || len(cb.Signature.OutputArgs) != 0 {
			return errors.Newf("invalid signature '%s'", cb.Signature)
		}
		if cb.Interval <= 0 {
			return errors.Newf("invalid interval '%v'", cb.Interval)
		}
		if cb.TimeBudget < 0 {
			return errors.Newf("invalid time budget '%v'", cb.TimeBudget)
		}
	}

	if strings.Contains(cb.Description, "`") {
		return errors.New("backquote character not allowed")
	}
	if strings.Contains(cb.Path, "`") {
		return errors.New("backquote character not allowed")
	}
	if cb.Queue != "" && cb.Queue != "default" && cb.Queue != "none" {
		return errors.Newf("invalid queue '%s'", cb.Queue)
	}
	return nil
}
