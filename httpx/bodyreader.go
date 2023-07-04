/*
Copyright (c) 2022-2023 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package httpx

import (
	"bytes"
	"io"
)

// BodyReader is used to wrap bytes in a closer+reader
// while allowing access to the underlying bytes.
type BodyReader struct {
	bytes  []byte
	reader io.Reader
}

// NewBodyReader creates a new closer+reader for the request body
// while allowing access to the underlying bytes.
func NewBodyReader(b []byte) *BodyReader {
	return &BodyReader{
		bytes:  b,
		reader: bytes.NewReader(b),
	}
}

// Read implements the io.Reader interface.
func (br *BodyReader) Read(p []byte) (n int, err error) {
	return br.reader.Read(p)
}

// Read implements the io.Closer interface.
func (br *BodyReader) Close() error {
	return nil
}

// Bytes gives access to the underlying bytes.
func (br *BodyReader) Bytes() []byte {
	return br.bytes
}

// Reset resets the underlying reader to the beginning.
func (br *BodyReader) Reset() {
	br.reader = bytes.NewReader(br.bytes)
}
