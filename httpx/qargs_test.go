/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package httpx

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHttpx_QArgs(t *testing.T) {
	assert.Equal(t, "b=true&i=123&s=String", QArgs{
		"s": "String",
		"i": 123,
		"b": true,
	}.Encode())
	assert.Equal(t, "b=true&i=123&s=String", QArgs{
		"s": "String",
		"i": 123,
		"b": true,
	}.String())
	assert.Equal(t,
		url.Values{
			"s": []string{"String"},
			"i": []string{"123"},
			"b": []string{"true"},
		},
		QArgs{
			"s": "String",
			"i": 123,
			"b": true,
		}.URLValues())
}
