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
	result := resolvedURL.String()
	result = strings.ReplaceAll(result, "%7B", "{")
	result = strings.ReplaceAll(result, "%7D", "}")
	return result, nil
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

// InjectPathArguments fills URL path arguments such as {arg} from the named value.
// If the argument is not named, e.g. {}, then a default name of path1, path2, etc. is assumed.
func InjectPathArguments(u string, values map[string]any) string {
	if !strings.Contains(u, "{") || !strings.Contains(u, "}") {
		return u
	}
	parts := strings.Split(u, "/")
	argIndex := 0
	for i := range parts {
		if !strings.HasPrefix(parts[i], "{") || !strings.HasSuffix(parts[i], "}") {
			continue
		}
		greedy := strings.HasSuffix(parts[i], "+}")
		argIndex++
		parts[i] = strings.TrimPrefix(parts[i], "{")
		parts[i] = strings.TrimSuffix(parts[i], "}")
		parts[i] = strings.TrimSuffix(parts[i], "+")
		if parts[i] == "" {
			parts[i] = fmt.Sprintf("path%d", argIndex)
		}
		if v, ok := values[parts[i]]; ok {
			parts[i] = url.PathEscape(fmt.Sprintf("%v", v))
			if greedy {
				// Allow slashes in greedy arguments
				parts[i] = strings.ReplaceAll(parts[i], "%2F", "/")
			}
		} else {
			parts[i] = ""
		}
	}
	return strings.Join(parts, "/")
}

// ResolvePathArguments transfers query arguments into path arguments, if present.
func ResolvePathArguments(u string) (resolved string, err error) {
	if !strings.Contains(u, "{") || !strings.Contains(u, "}") {
		return u, nil
	}
	var query url.Values
	u, q, found := strings.Cut(u, "?")
	if found {
		query, err = url.ParseQuery(q)
		if err != nil {
			return "", errors.Trace(err)
		}
	}
	parts := strings.Split(u, "/")
	argIndex := 0
	for i := range parts {
		if !strings.HasPrefix(parts[i], "{") || !strings.HasSuffix(parts[i], "}") {
			continue
		}
		greedy := strings.HasSuffix(parts[i], "+}")
		argIndex++
		parts[i] = strings.TrimPrefix(parts[i], "{")
		parts[i] = strings.TrimSuffix(parts[i], "}")
		parts[i] = strings.TrimSuffix(parts[i], "+")
		if parts[i] == "" {
			parts[i] = fmt.Sprintf("path%d", argIndex)
		}
		if vv, ok := query[parts[i]]; ok && len(vv) > 0 {
			delete(query, parts[i])
			v := vv[len(vv)-1]
			parts[i] = url.PathEscape(fmt.Sprintf("%v", v))
			if greedy {
				// Allow slashes in greedy arguments
				parts[i] = strings.ReplaceAll(parts[i], "%2F", "/")
			}
		} else {
			parts[i] = ""
		}
	}
	u = strings.Join(parts, "/")
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
	return u, nil
}
