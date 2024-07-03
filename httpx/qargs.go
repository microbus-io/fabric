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

package httpx

import (
	"fmt"
	"net/url"
)

// Ensure interface
var _ = fmt.Stringer(QArgs{})

/*
QArgs faciliates the creation of URL query arguments for the common case where there is only
one value per key. Values of any type are converted to a string. Keys are case-sensitive.

Usage:

	u := "https://example.com/path?"+QArgs{
		"hello": "World",
		"number": 6,
		"id", key
	}.Encode()
*/
type QArgs map[string]any

// URLValues generates a standard URL values map.
func (q QArgs) URLValues() url.Values {
	vals := url.Values{}
	for k, v := range q {
		vals[k] = []string{
			fmt.Sprintf("%v", v),
		}
	}
	return vals
}

// Encode encodes the values into “URL encoded” form ("bar=baz&foo=quux") sorted by key.
func (q QArgs) Encode() string {
	return q.URLValues().Encode()
}

// String encodes the values into “URL encoded” form ("bar=baz&foo=quux") sorted by key.
func (q QArgs) String() string {
	return q.URLValues().Encode()
}
