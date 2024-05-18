/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package httpx

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/microbus-io/fabric/errors"
)

// JoinHostAndPath combines the path shorthand with a host name.
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

// PrepareQueryString composes a query string from a list of key-value pairs, making sure to encode appropriately.
// The arguments are sorted by key.
func PrepareQueryString(kvPairs ...any) string {
	vals := url.Values{}
	for i := 0; i < len(kvPairs); i += 2 {
		k := fmt.Sprintf("%v", kvPairs[i])
		v := fmt.Sprintf("%v", kvPairs[i+1])
		vals[k] = []string{v}
	}
	return vals.Encode()
}
