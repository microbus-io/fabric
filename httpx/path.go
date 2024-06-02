/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package httpx

import (
	"net/url"
	"strings"

	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/utils"
)

// JoinHostAndPath combines the path shorthand with a hostname.
func JoinHostAndPath(host string, path string) string {
	if path == "" {
		// (empty)
		return "https://" + host + ":443"
	}
	if strings.HasPrefix(path, ":") {
		// :1080/path
		return "https://" + host + path
	}
	if strings.HasPrefix(path, "//") {
		// //host.name/path/with/slash
		return "https:" + path
	}
	if strings.HasPrefix(path, "/") {
		// /path/with/slash
		return "https://" + host + ":443" + path
	}
	if !strings.Contains(path, "://") {
		// path/with/no/slash
		return "https://" + host + ":443/" + path
	}
	return path
}

// ResolveURL resolves a URL in relation to the endpoint's base path.
func ResolveURL(base string, relative string) (resolved string, err error) {
	if relative == "" {
		return base, nil
	}
	baseURL, err := url.Parse(base)
	if err != nil {
		return "", errors.Trace(err)
	}
	relativeURL, err := url.Parse(relative)
	if err != nil {
		return "", errors.Trace(err)
	}
	resolvedURL := baseURL.ResolveReference(relativeURL)
	return resolvedURL.String(), nil
}

// ParseURL returns a canonical version of the parsed URL with the scheme and port filled in if omitted.
func ParseURL(rawURL string) (canonical *url.URL, err error) {
	if strings.Contains(rawURL, "`") {
		return nil, errors.New("backquote not allowed")
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, errors.Trace(err)
	}
	if err := utils.ValidateHostname(parsed.Hostname()); err != nil {
		return nil, errors.Trace(err)
	}
	if parsed.Scheme == "" {
		parsed.Scheme = "https"
	}
	if parsed.Port() == "" {
		port := "443"
		if parsed.Scheme == "http" {
			port = "80"
		}
		parsed.Host += ":" + port
	}
	return parsed, nil
}
