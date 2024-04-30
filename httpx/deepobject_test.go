/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package httpx

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestHttpx_DeepObject(t *testing.T) {
	type Point struct {
		X int
		Y int
	}
	type Doc struct {
		I       int       `json:"i"`
		Zero    int       `json:"z,omitempty"`
		B       bool      `json:"b"`
		F       float32   `json:"f"`
		S       string    `json:"s"`
		Pt      Point     `json:"pt"`
		Empty   *Point    `json:"e,omitempty"`
		Null    *Point    `json:"n"`
		Special string    `json:"sp"`
		T       time.Time `json:"t"`
	}

	// Encode
	d1 := Doc{
		I:       5,
		B:       true,
		F:       5.67,
		S:       "Hello",
		Special: "Q&A",
		Pt:      Point{X: 3, Y: 4},
		T:       time.Date(2001, 10, 1, 12, 0, 0, 0, time.UTC),
	}
	values, err := EncodeDeepObject(d1)
	if assert.NoError(t, err) {
		assert.Equal(t, "5", values.Get("i"))
		assert.Equal(t, "true", values.Get("b"))
		assert.Equal(t, "5.67", values.Get("f"))
		assert.Equal(t, "Hello", values.Get("s"))
		assert.Equal(t, "Q&A", values.Get("sp"))
		assert.Equal(t, "3", values.Get("pt[X]"))
		assert.Equal(t, "4", values.Get("pt[Y]"))
		assert.Equal(t, "2001-10-01T12:00:00Z", values.Get("t"))
	}

	var d2 Doc
	err = DecodeDeepObject(values, &d2)
	if assert.NoError(t, err) {
		assert.Equal(t, d1, d2)
	}
}
