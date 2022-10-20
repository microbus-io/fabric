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

// ValidateURL indicates if the URL has a valid host name and port.
func ValidateURL(u string) error {
	parsed, err := url.Parse(u)
	if err != nil {
		return errors.Trace(err)
	}

	// Host
	if err := ValidateHostName(parsed.Hostname()); err != nil {
		return errors.Trace(err)
	}

	// Port
	port := 443
	if parsed.Port() != "" {
		port, err = strconv.Atoi(parsed.Port())
		if err != nil {
			return errors.Newf("invalid port '%s'", parsed.Port())
		}
	}
	if port < 0 || port > 65535 {
		return errors.Newf("invalid port '%d'", port)
	}

	return nil
}
