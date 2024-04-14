/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package httpx

import (
	"bytes"
	"io"
	"net/http"
	"strconv"

	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/frame"
)

// FragRequest transforms an HTTP request into one or more fragments that do not exceed a given size.
// Fragmenting is needed because NATS imposes a maximum size for messages
type FragRequest struct {
	origRequest   *http.Request
	bodyFragments [][]byte
	noFrags       bool
}

// NewFragRequest creates a new request fragmentor
func NewFragRequest(r *http.Request, fragmentSize int64) (*FragRequest, error) {
	if r.Body == nil {
		return &FragRequest{origRequest: r, noFrags: true}, nil
	}

	result := &FragRequest{origRequest: r}

	if bodyReader, ok := (r.Body).(*BodyReader); ok {
		// BodyReader optimization
		body := bodyReader.Bytes()
		if len(body) <= int(fragmentSize) {
			r.Header.Set("Content-Length", strconv.Itoa(len(body)))
			return &FragRequest{origRequest: r, noFrags: true}, nil
		}
		for s := int64(0); s < int64(len(body)); s += fragmentSize {
			if s+fragmentSize < int64(len(body)) {
				result.bodyFragments = append(result.bodyFragments, body[s:s+fragmentSize])
			} else {
				result.bodyFragments = append(result.bodyFragments, body[s:])
			}
		}
	} else {
		// Any reader
		for {
			var buf bytes.Buffer
			lr := io.LimitReader(r.Body, int64(fragmentSize))
			n, err := io.Copy(&buf, lr)
			if err != nil {
				return nil, errors.Trace(err)
			}
			result.bodyFragments = append(result.bodyFragments, buf.Bytes())
			if n < fragmentSize {
				break
			}
		}
	}

	return result, nil
}

// N is the number of fragments
func (fr *FragRequest) N() int {
	if fr.noFrags {
		return 1
	}
	return len(fr.bodyFragments)
}

// Fragment returns the 1-indexed fragment
func (fr *FragRequest) Fragment(index int) (f *http.Request, err error) {
	if fr.noFrags {
		if index == 1 {
			return fr.origRequest, nil
		}
		return nil, errors.New("index out of bounds")
	}

	if index < 1 || index > len(fr.bodyFragments) {
		return nil, errors.New("index out of bounds")
	}
	body := fr.bodyFragments[index-1]
	n := int64(len(body))

	// Prepare the HTTP request
	fragment, err := http.NewRequest(fr.origRequest.Method, fr.origRequest.URL.String(), bytes.NewReader(body))
	if err != nil {
		return nil, errors.Trace(err)
	}
	for k, vv := range fr.origRequest.Header {
		for _, v := range vv {
			fragment.Header.Set(k, v)
		}
	}
	fragment.Header.Set("Content-Length", strconv.FormatInt(n, 10))
	frame.Of(fragment).SetFragment(index, len(fr.bodyFragments))

	return fragment, nil
}
