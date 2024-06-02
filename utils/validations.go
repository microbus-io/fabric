/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package utils

import (
	"regexp"
	"strings"

	"github.com/microbus-io/fabric/errors"
)

var (
	hostnameRegexp = regexp.MustCompile(`^[a-z0-9_\-]+(\.[a-z0-9_\-]+)*$`)
	configRegexp   = regexp.MustCompile(`^[a-z][a-z0-9]*$`)
	tickerRegexp   = regexp.MustCompile(`^[a-z][a-z0-9]*$`)
)

// ValidateHostname indicates if the hostname is a valid microservice hostname.
// Hostnames must contain only alphanumeric characters, hyphens, underscores and dot separators.
func ValidateHostname(hostname string) error {
	if !hostnameRegexp.MatchString(strings.ToLower(hostname)) {
		// The hostname "all" is reserved to refer to all microservices
		return errors.Newf("invalid hostname '%s'", hostname)
	}
	return nil
}

// ValidateConfigName indicates if the name can be used for a config.
// Config names must start with a letter and contain only alphanumeric characters.
func ValidateConfigName(name string) error {
	if !configRegexp.MatchString(strings.ToLower(name)) {
		return errors.Newf("invalid config name '%s'", name)
	}
	return nil
}

// ValidateTickerName indicates if the name can be used for a ticker.
// Ticker names must start with a letter and contain only alphanumeric characters.
func ValidateTickerName(name string) error {
	if !tickerRegexp.MatchString(strings.ToLower(name)) {
		return errors.Newf("invalid ticker name '%s'", name)
	}
	return nil
}
