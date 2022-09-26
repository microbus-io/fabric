package sub

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/microbus-io/fabric/errors"
	"github.com/nats-io/nats.go"
)

// HTTPHandler extends the standard Go's http.Handler with an error
type HTTPHandler func(w http.ResponseWriter, r *http.Request) error

// Subscription handles incoming requests
type Subscription struct {
	Host    string
	Port    int
	Path    string
	Handler HTTPHandler
	NATSSub *nats.Subscription
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
func NewSub(defaultHost string, path string) (*Subscription, error) {
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
			return nil, errors.Newf("invalid port: %s", u.Port())
		}
	}
	if port < 0 || port > 65535 {
		return nil, errors.Newf("invalid port: %d", port)
	}

	return &Subscription{
		Host: u.Hostname(),
		Port: port,
		Path: u.Path,
	}, nil
}
