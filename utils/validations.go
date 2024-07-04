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

package utils

import (
	"regexp"
	"strings"

	"github.com/microbus-io/fabric/errors"
)

var (
	hostnameValidator = regexp.MustCompile(`^[a-z0-9_\-]+(\.[a-z0-9_\-]+)*$`)
	configValidator   = regexp.MustCompile(`^[a-z][a-z0-9]*$`)
	tickerValidator   = regexp.MustCompile(`^[a-z][a-z0-9]*$`)
)

// ValidateHostname indicates if the hostname is a valid microservice hostname.
// Hostnames must contain only alphanumeric characters, hyphens, underscores and dot separators.
func ValidateHostname(hostname string) error {
	if !hostnameValidator.MatchString(strings.ToLower(hostname)) {
		return errors.Newf("invalid hostname '%s'", hostname)
	}
	return nil
}

// ValidateConfigName indicates if the name can be used for a config.
// Config names must start with a letter and contain only alphanumeric characters.
func ValidateConfigName(name string) error {
	if !configValidator.MatchString(strings.ToLower(name)) {
		return errors.Newf("invalid config name '%s'", name)
	}
	return nil
}

// ValidateTickerName indicates if the name can be used for a ticker.
// Ticker names must start with a letter and contain only alphanumeric characters.
func ValidateTickerName(name string) error {
	if !tickerValidator.MatchString(strings.ToLower(name)) {
		return errors.Newf("invalid ticker name '%s'", name)
	}
	return nil
}
