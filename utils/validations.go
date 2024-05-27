/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package utils

import (
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/microbus-io/fabric/errors"
)

// ValidateHostname indicates if the hostname is a valid microservice hostname.
// Hostnames must only contain alphanumeric characters, hyphens, underscores and dot separators.
func ValidateHostname(hostname string) error {
	hn := strings.ToLower(hostname)
	match, _ := regexp.MatchString(`^[a-z0-9_\-]+(\.[a-z0-9_\-]+)*$`, hn)
	if !match {
		// The hostname "all" is reserved to refer to all microservices
		return errors.Newf("invalid hostname '%s'", hostname)
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
// It returns an error if the URL has a invalid scheme, hostname or port.
func ParseURL(u string) (canonical *url.URL, err error) {
	if strings.Contains(u, "`") {
		return nil, errors.New("backquote not allowed")
	}

	parsed, err := url.Parse(u)
	if err != nil {
		return nil, errors.Trace(err)
	}

	// Host
	if err := ValidateHostname(parsed.Hostname()); err != nil {
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
