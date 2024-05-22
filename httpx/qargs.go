/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
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
