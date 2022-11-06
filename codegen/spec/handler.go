package spec

import (
	"regexp"
	"strings"
	"time"

	"github.com/microbus-io/fabric/cfg"
	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/sub"
	"github.com/microbus-io/fabric/utils"
)

// Handler is the spec of a callback handler.
// Web requests, lifecycle events, config changes, tickers, etc.
type Handler struct {
	Type   string `yaml:"-"`
	Exists bool   `yaml:"-"`

	// Shared
	Signature   *Signature `yaml:"signature"`
	Description string     `yaml:"description"`
	Path        string     `yaml:"path"`
	Queue       string     `yaml:"queue"`

	// Sink
	Event   string `yaml:"event"`
	Source  string `yaml:"source"`
	ForHost string `yaml:"forHost"`

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
	h.Queue = strings.ToLower(h.Queue)

	if h.Event == "" {
		h.Event = h.Name()
	}

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
		return errors.Newf("invalid queue '%s' in '%s'", h.Queue, h.Name())
	}
	if strings.Contains(h.Path, "`") {
		return errors.Newf("backquote not allowed in path '%s' in '%s'", h.Path, h.Name())
	}
	joined := sub.JoinHostAndPath("example.com", h.Path)
	_, err := utils.ParseURL(joined)
	if err != nil {
		return errors.Newf("invalid path '%s' in '%s'", h.Path, h.Name())
	}
	if h.Validation != "" {
		_, err := cfg.NewConfig(
			h.Name(),
			cfg.Validation(h.Validation),
			cfg.DefaultValue(h.Default),
		)
		if err != nil {
			return errors.Trace(err)
		}
	}

	// Type will be empty during initial parsing.
	// It will get filled by the parent service which will then call this method again.
	if h.Type == "" {
		return nil
	}

	h.Description = conformDesc(
		h.Description,
		h.Name()+" is a "+h.Type+".",
		h.Name(),
	)

	// Validate types and number of arguments
	switch h.Type {
	case "web":
		if len(h.Signature.InputArgs) != 0 || len(h.Signature.OutputArgs) != 0 {
			return errors.Newf("arguments or return values not allowed in '%s'", h.Signature.OrigString)
		}
	case "config":
		if len(h.Signature.InputArgs) != 0 {
			return errors.Newf("arguments not allowed in '%s'", h.Signature.OrigString)
		}
		if len(h.Signature.OutputArgs) != 1 {
			return errors.Newf("single return value expected in '%s'", h.Signature.OrigString)
		}
		t := h.Signature.OutputArgs[0].Type
		if t != "string" && t != "int" && t != "bool" && t != "time.Duration" && t != "float64" {
			return errors.Newf("invalid return type '%s' in '%s'", t, h.Signature.OrigString)
		}
	case "ticker":
		if len(h.Signature.InputArgs) != 0 || len(h.Signature.OutputArgs) != 0 {
			return errors.Newf("arguments or return values not allowed in '%s'", h.Signature.OrigString)
		}
		if h.Interval <= 0 {
			return errors.Newf("non-positive interval '%v' in '%s'", h.Interval, h.Name())
		}
		if h.TimeBudget < 0 {
			return errors.Newf("negative time budget '%v' in '%s'", h.TimeBudget, h.Name())
		}
	case "function", "event", "sink":
		argNames := map[string]bool{}
		for _, arg := range h.Signature.InputArgs {
			if argNames[arg.Name] {
				return errors.Newf("duplicate arg name '%s' in '%s'", arg.Name, h.Signature.OrigString)
			}
			argNames[arg.Name] = true
		}
		for _, arg := range h.Signature.OutputArgs {
			if argNames[arg.Name] {
				return errors.Newf("duplicate arg name '%s' in '%s'", arg.Name, h.Signature.OrigString)
			}
			argNames[arg.Name] = true
		}
	}

	startsWithOn := regexp.MustCompile(`^On[A-Z][a-zA-Z0-9]*$`)
	if h.Type == "sink" {
		match, _ := regexp.MatchString(`^[a-z][a-zA-Z0-9\.\-]*(/[a-z][a-zA-Z0-9\.\-]*)*$`, h.Source)
		if !match {
			return errors.Newf("invalid source '%s' in '%s'", h.Source, h.Name())
		}
		if h.ForHost != "" {
			err := utils.ValidateHostName(h.ForHost)
			if err != nil {
				return errors.Newf("invalid host name '%s' in '%s'", h.ForHost, h.Name())
			}
		}
		if !startsWithOn.MatchString(h.Event) {
			return errors.Newf("event name '%s' must start with 'On' in '%s'", h.Event, h.Name())
		}
	}
	if h.Type == "sink" || h.Type == "event" {
		if !startsWithOn.MatchString(h.Name()) {
			return errors.Newf("function name must start with 'On' in '%s'", h.Signature.OrigString)
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
	b.WriteString("ctx context.Context")
	for _, arg := range h.Signature.InputArgs {
		b.WriteString(", ")
		b.WriteString(arg.Name)
		b.WriteString(" ")
		b.WriteString(arg.Type)
	}
	return b.String()
}

// In returns the output argument list as a string.
func (h *Handler) Out() string {
	var b strings.Builder
	for _, arg := range h.Signature.OutputArgs {
		b.WriteString(arg.Name)
		b.WriteString(" ")
		b.WriteString(arg.Type)
		b.WriteString(", ")
	}
	b.WriteString("err error")
	return b.String()
}

// SourceSuffix returns the last piece of the by definition,
// which is expected to point to a microservice.
func (h *Handler) SourceSuffix() string {
	p := strings.LastIndex(h.Source, "/")
	if p < 0 {
		p = -1
	}
	return h.Source[p+1:]
}
