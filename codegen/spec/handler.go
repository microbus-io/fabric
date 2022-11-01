package spec

import (
	"strings"
	"time"

	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/sub"
	"github.com/microbus-io/fabric/utils"
)

// Handler is the spec of a callback handler.
// Web requests, lifecycle events, config changes, tickers, etc.
type Handler struct {
	Type   string `yaml:"-"`
	Exists bool   `yaml:"-"`

	// Web or Function
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
func (h *Handler) UnmarshalYAML(unmarshal func(interface{}) error) error {
	// Unmarshal
	type different Handler
	var x different
	err := unmarshal(&x)
	if err != nil {
		return errors.Trace(err)
	}
	*h = Handler(x)

	// Post processing
	if h.Path == "" {
		h.Path = "/" + kebabCase(h.Name())
	}
	if h.Path == "^" {
		h.Path = ""
	}
	h.Path = strings.Replace(h.Path, "...", kebabCase(h.Name()), 1)
	h.Description = conformDesc(
		h.Description,
		h.Name()+" is a "+h.Type+" handler.",
		h.Name(),
	)
	h.Queue = strings.ToLower(h.Queue)

	// Validate
	err = h.validate()
	if err != nil {
		return errors.Trace(err)
	}
	return nil
}

// validate validates the data after unmarshaling.
func (h *Handler) validate() error {
	if h.Queue != "" && h.Queue != "default" && h.Queue != "none" {
		return errors.Newf("invalid queue '%s'", h.Queue)
	}
	if strings.Contains(h.Path, "`") {
		return errors.Newf("backquote not allowed in path '%s'", h.Path)
	}
	joined := sub.JoinHostAndPath("example.com", h.Path)
	_, err := utils.ParseURL(joined)
	if err != nil {
		return errors.Newf("invalid path '%s'", h.Path)
	}

	// Type will be empty during initial parsing
	switch h.Type {
	case "web":
		if len(h.Signature.InputArgs) != 0 || len(h.Signature.OutputArgs) != 0 {
			return errors.Newf("invalid signature '%s'", h.Signature.OrigString)
		}
	case "config":
		if len(h.Signature.InputArgs) != 0 || len(h.Signature.OutputArgs) != 1 {
			return errors.Newf("invalid signature '%s'", h.Signature.OrigString)
		}
	case "ticker":
		if len(h.Signature.InputArgs) != 0 || len(h.Signature.OutputArgs) != 0 {
			return errors.Newf("invalid signature '%s'", h.Signature.OrigString)
		}
		if h.Interval <= 0 {
			return errors.Newf("invalid interval '%v'", h.Interval)
		}
		if h.TimeBudget < 0 {
			return errors.Newf("invalid time budget '%v'", h.TimeBudget)
		}
	case "function":
		argNames := map[string]bool{}
		for _, arg := range h.Signature.InputArgs {
			if argNames[arg.Name] {
				return errors.Newf("duplicate arg name '%s'", h.Signature.OrigString)
			}
			argNames[arg.Name] = true
		}
		for _, arg := range h.Signature.OutputArgs {
			if argNames[arg.Name] {
				return errors.Newf("duplicate arg name '%s'", h.Signature.OrigString)
			}
			argNames[arg.Name] = true
		}
	}

	return nil
}

// Name of the handler function.
func (h *Handler) Name() string {
	return h.Signature.Name
}

// In returns the input argument list as a string.
func (h *Handler) In() string {
	var b strings.Builder
	for i, arg := range h.Signature.InputArgs {
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
func (h *Handler) Out() string {
	var b strings.Builder
	for i, arg := range h.Signature.OutputArgs {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(arg.Name)
		b.WriteString(" ")
		b.WriteString(arg.Type)
	}
	return b.String()
}
