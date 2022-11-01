package sub

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/utils"
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
	Handler   any
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
func NewSub(defaultHost string, path string, handler HTTPHandler, options ...Option) (*Subscription, error) {
	joined := JoinHostAndPath(defaultHost, path)
	u, err := utils.ParseURL(joined)
	if err != nil {
		return nil, errors.Trace(err)
	}
	port, _ := strconv.Atoi(u.Port())
	sub := &Subscription{
		Host:    u.Hostname(),
		Port:    port,
		Path:    u.Path,
		Queue:   defaultHost,
		Handler: handler,
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
	return fmt.Sprintf("%s:%d%s", sub.Host, sub.Port, sub.Path)
}

// JoinHostAndPath combines the path shorthand with a host name.
func JoinHostAndPath(host string, path string) (joined string) {
	joined = path
	if path == "" {
		// (empty)
		joined = "https://" + host + ":443"
	} else if strings.HasPrefix(path, ":") {
		// :1080/path
		joined = "https://" + host + path
	} else if strings.HasPrefix(path, "/") {
		// /path/with/slash
		joined = "https://" + host + ":443" + path
	} else if !strings.Contains(path, "://") {
		// path/with/no/slash
		joined = "https://" + host + ":443/" + path
	}
	return joined
}
