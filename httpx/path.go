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
