/*
Copyright (c) 2022-2023 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package httpx

import "strings"

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
