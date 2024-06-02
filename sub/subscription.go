/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package sub

import (
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/httpx"
	"github.com/nats-io/nats.go"
)

var methodValidator = regexp.MustCompile(`^[A-Z]+$`)

// HTTPHandler extends the standard http.Handler to also return an error.
type HTTPHandler func(w http.ResponseWriter, r *http.Request) error

// Subscription handles incoming requests.
// Although technically public, it is used internally and should not be constructed by microservices directly.
type Subscription struct {
	Host      string
	Port      string
	Method    string
	Path      string
	Queue     string
	Handler   any
	HostSub   *nats.Subscription
	DirectSub *nats.Subscription
}

/*
NewSub creates a new subscription for the indicated path.
If the path does not include a hostname, the microservice's default hostname is used.
If a port is not specified, 443 is used by default. Port 0 is used to designate any port.
The subscription can be limited to one HTTP method such as "GET" or "POST",
or an asterisk "*" can be used to accept any method.

Examples of valid paths:

	(empty)
	/
	/path
	:1080
	:1080/
	:1080/path
	:0/any/port
	/path/with/slash
	path/with/no/slash
	https://www.example.com/path
	https://www.example.com:1080/path
	//www.example.com:1080/path
*/
func NewSub(method string, defaultHost string, path string, handler HTTPHandler, options ...Option) (*Subscription, error) {
	joined := httpx.JoinHostAndPath(defaultHost, path)
	u, err := httpx.ParseURL(joined)
	if err != nil {
		return nil, errors.Trace(err)
	}
	_, err = strconv.Atoi(u.Port())
	if err != nil {
		return nil, errors.Trace(err)
	}
	method = strings.ToUpper(method)
	if method != "*" && !methodValidator.MatchString(method) {
		return nil, errors.Newf("invalid method '%s'", method)
	}
	sub := &Subscription{
		Host:    u.Hostname(),
		Port:    u.Port(),
		Method:  method,
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

// Apply the provided options to the subscription.
func (sub *Subscription) Apply(options ...Option) error {
	for _, opt := range options {
		err := opt(sub)
		if err != nil {
			return errors.Trace(err)
		}
	}
	return nil
}

// Canonical returns the fully-qualified canonical host:port/path of the subscription, not including the scheme.
func (sub *Subscription) Canonical() string {
	return fmt.Sprintf("%s:%s%s", sub.Host, sub.Port, sub.Path)
}
