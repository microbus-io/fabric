package configurator

import (
	"strings"

	"github.com/microbus-io/fabric/errors"
	"gopkg.in/yaml.v2"
)

type repository struct {
	values map[string]map[string]string // host name -> config property name -> value
}

/*
LoadYAML loads the values specified in the YAML into the repo.
The expected format of the YAML is:

	hello.example:
	  greeting: Ciao
	  repeat: 3
	http.ingress.sys:
	  port: 9090
	all:
	  sql: sql.host

The scope argument will load only properties that are listed under that domain or a subdomain thereof.
An empty value of scope or the magic keyword "all" will load values of all domains.
*/
func (r *repository) LoadYAML(data []byte, scope string) error {
	var values map[string]map[string]string
	err := yaml.Unmarshal(data, &values)
	if err != nil {
		return errors.Trace(err)
	}
	scope = strings.TrimSpace(strings.ToLower(scope))

	if r.values == nil {
		r.values = map[string]map[string]string{}
	}
	for domain, valmap := range values {
		domain = strings.TrimSpace(strings.ToLower(domain))
		if r.values[domain] == nil {
			r.values[domain] = map[string]string{}
		}
		if scope == "" || scope == "all" || scope == domain || strings.HasSuffix(domain, "."+scope) {
			for name, val := range valmap {
				name = strings.TrimSpace(strings.ToLower(name))
				r.values[domain][name] = val
			}
		}
	}
	return nil
}

// Value returns the value most specifically associated with the property name.
// A value set for domain "www.example.com" is more specific than one set for domain "example.com"
// which is more specific than one set for domain "com" which is more specific than one set for domain "all".
func (r *repository) Value(host string, name string) (value string, ok bool) {
	if r.values == nil {
		return "", false
	}
	if r.values["all"] != nil {
		value, ok = r.values["all"][name]
	}
	host = strings.TrimSpace(strings.ToLower(host))
	name = strings.TrimSpace(strings.ToLower(name))
	segments := strings.Split(host, ".")
	for i := len(segments) - 1; i >= 0; i-- {
		domain := strings.Join(segments[i:], ".")
		if r.values[domain] != nil {
			if v, found := r.values[domain][name]; found {
				value, ok = v, true
			}
		}
	}
	return value, ok
}
