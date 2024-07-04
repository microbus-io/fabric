/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
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
	"github.com/microbus-io/fabric/utils"
	"github.com/nats-io/nats.go"
)

var methodValidator = regexp.MustCompile(`^[A-Z]+$`)

// HTTPHandler extends the standard http.Handler to also return an error.
type HTTPHandler func(w http.ResponseWriter, r *http.Request) error

// Subscription handles incoming requests.
// Although technically public, it is used internally and should not be constructed by microservices directly.
type Subscription struct {
	Host     string
	Port     string
	Method   string
	Path     string
	Queue    string
	Handler  any
	Subs     []*nats.Subscription
	specPath string
}

/*
NewSub creates a new subscription for the indicated path.
If the path does not include a hostname, the microservice's default hostname is used.
If a port is not specified, 443 is used by default. Port 0 is used to designate any port.
The subscription can be limited to one HTTP method such as "GET", "POST", etc.
or "ANY" can be used to accept any method.

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
	if !methodValidator.MatchString(method) {
		return nil, errors.Newf("invalid method '%s'", method)
	}
	parts := strings.Split(u.Path, "/")
	for i := range parts {
		open := strings.Index(parts[i], "{")
		if open > 0 {
			return nil, errors.Newf("path argument '%s' must span entire section", parts[i])
		}
		close := strings.LastIndex(parts[i], "}")
		if open == -1 && close == -1 {
			continue
		}
		if close <= open || open == -1 {
			return nil, errors.Newf("malformed path argument '%s'", parts[i])
		}
		if close < len(parts[i])-1 {
			return nil, errors.Newf("path argument '%s' must span entire section", parts[i])
		}
		name := parts[i]
		name = strings.TrimPrefix(name, "{")
		name = strings.TrimSuffix(name, "}")
		if strings.HasSuffix(name, "+") && i != len(parts)-1 {
			return nil, errors.Newf("greedy path argument '%s' must end path", parts[i])
		}
		name = strings.TrimSuffix(name, "+")
		if name != "" && !utils.IsLowerCaseIdentifier(name) {
			return nil, errors.Newf("name of path argument '%s' must be an identifier", parts[i])
		}
	}
	sub := &Subscription{
		Host:     u.Hostname(),
		Port:     u.Port(),
		Method:   method,
		Path:     u.Path,
		Queue:    defaultHost,
		Handler:  handler,
		specPath: path,
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

// RefreshHostname refreshes the subscription for a different hostname.
func (sub *Subscription) RefreshHostname(defaultHost string) error {
	joined := httpx.JoinHostAndPath(defaultHost, sub.specPath)
	u, err := httpx.ParseURL(joined)
	if err != nil {
		return errors.Trace(err)
	}
	sub.Host = u.Hostname()
	return nil
}
