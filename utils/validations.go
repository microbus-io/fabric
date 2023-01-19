/*
Copyright 2023 Microbus LLC and various contributors

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

package utils

import (
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/microbus-io/fabric/errors"
)

// ValidateHostName indicates if the host name is a valid microservice host name.
// Host names must only contain alphanumeric characters and the dot separator.
// Hyphens or underlines are not allowed.
func ValidateHostName(hostName string) error {
	hn := strings.ToLower(hostName)
	match, _ := regexp.MatchString(`^[a-z0-9]+(\.[a-z0-9]+)*$`, hn)
	if !match {
		// The hostname "all" is reserved to refer to all microservices
		return errors.Newf("invalid host name '%s'", hostName)
	}
	return nil
}

// ValidateConfigName indicates if the name can be used for a config.
// Config names must start with a letter and contain only alphanumeric characters.
func ValidateConfigName(name string) error {
	n := strings.ToLower(name)
	match, _ := regexp.MatchString(`^[a-z][a-z0-9]*$`, n)
	if !match {
		return errors.Newf("invalid config name '%s'", name)
	}
	return nil
}

// ValidateTickerName indicates if the name can be used for a ticker.
// Ticker names must start with a letter and contain only alphanumeric characters.
func ValidateTickerName(name string) error {
	n := strings.ToLower(name)
	match, _ := regexp.MatchString(`^[a-z][a-z0-9]*$`, n)
	if !match {
		return errors.Newf("invalid ticker name '%s'", name)
	}
	return nil
}

// ParseURL returns a canonical version of the parsed URL with the scheme and port filled in if omitted.
// It returns an error if the URL has a invalid scheme, host name or port.
func ParseURL(u string) (canonical *url.URL, err error) {
	if strings.Contains(u, "`") {
		return nil, errors.New("backquote not allowed")
	}

	parsed, err := url.Parse(u)
	if err != nil {
		return nil, errors.Trace(err)
	}

	// Host
	if err := ValidateHostName(parsed.Hostname()); err != nil {
		return nil, errors.Trace(err)
	}

	// Scheme
	if parsed.Scheme == "" {
		parsed.Scheme = "https"
	}

	// Port
	port := 443
	if parsed.Scheme == "http" {
		port = 80
	}
	if parsed.Port() != "" {
		port, err = strconv.Atoi(parsed.Port())
		if err != nil {
			return nil, errors.Newf("invalid port '%s'", parsed.Port())
		}
	} else {
		parsed.Host += ":" + strconv.Itoa(port)
	}
	if port < 1 || port > 65535 {
		return nil, errors.Newf("invalid port '%d'", port)
	}

	return parsed, nil
}
