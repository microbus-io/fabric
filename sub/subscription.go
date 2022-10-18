package sub

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/microbus-io/fabric/errors"
	"github.com/nats-io/nats.go"
)

// HTTPHandler extends the standard http.Handler to also return an error
type HTTPHandler func(w http.ResponseWriter, r *http.Request) error

// Subscription handles incoming requests.
// Although technically public, it is used internally and should not be constructed by microservices directly
type Subscription struct {
	Host      string
	Port      int
	Path      string
	Queue     string
	Handler   HTTPHandler
	HostSub   *nats.Subscription
	DirectSub *nats.Subscription
}

/*
NewSub creates a new subscription for the indicated path.
If the path does not include a host name, the default host is used.
If a port is not specified, 443 is used by default.

Examples of valid paths:

	(empty)
	/
	:1080
	:1080/
	:1080/path
	/path/with/slash
	path/with/no/slash
	https://www.example.com/path
	https://www.example.com:1080/path
*/
func NewSub(defaultHost string, path string, options ...Option) (*Subscription, error) {
	spec := path
	if path == "" {
		// (empty)
		spec = "https://" + defaultHost + ":443"
	} else if strings.HasPrefix(path, ":") {
		// :1080/path
		spec = "https://" + defaultHost + path
	} else if strings.HasPrefix(path, "/") {
		// /path/with/slash
		spec = "https://" + defaultHost + ":443" + path
	} else if !strings.Contains(path, "://") {
		// path/with/no/slash
		spec = "https://" + defaultHost + ":443/" + path
	}

	u, err := url.Parse(spec)
	if err != nil {
		return nil, errors.Trace(err)
	}

	// Port
	port := 443
	if u.Port() != "" {
		port, err = strconv.Atoi(u.Port())
		if err != nil {
			return nil, errors.Newf("invalid port '%s'", u.Port())
		}
	}
	if port < 0 || port > 65535 {
		return nil, errors.Newf("invalid port '%d'", port)
	}

	sub := &Subscription{
		Host:  u.Hostname(),
		Port:  port,
		Path:  u.Path,
		Queue: defaultHost,
	}
	err = sub.Apply(options...)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return sub, nil
}

// Apply the provided options to the subscription
func (sub *Subscription) Apply(options ...Option) error {
	for _, opt := range options {
		err := opt(sub)
		if err != nil {
			return errors.Trace(err)
		}
	}
	return nil
}

// Canonical returns the fully-qualified canonical path of the subscription
func (sub *Subscription) Canonical() string {
	return fmt.Sprintf("https://%s:%d%s", sub.Host, sub.Port, sub.Path)
}
