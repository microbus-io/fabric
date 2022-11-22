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
