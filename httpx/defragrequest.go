/*
Copyright (c) 2023-2026 Microbus LLC and various contributors

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
	"io"
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/microbus-io/errors"
	"github.com/microbus-io/fabric/frame"
)

// DefragRequest merges together multiple fragments back into a single HTTP request
type DefragRequest struct {
	fragments    map[int]*http.Request
	maxIndex     int
	arrived      int
	lastActivity atomic.Int64
	mux          sync.Mutex
}

// NewDefragRequest creates a new request integrator.
func NewDefragRequest() *DefragRequest {
	st := &DefragRequest{
		fragments: map[int]*http.Request{},
	}
	st.lastActivity.Store(time.Now().UnixMilli())
	return st
}

// LastActivity indicates how long ago was the last fragment added.
func (st *DefragRequest) LastActivity() time.Duration {
	return time.Duration(time.Now().UnixMilli()-st.lastActivity.Load()) * time.Millisecond
}

// Integrated returns all the fragments have been collected as a single HTTP request.
func (st *DefragRequest) Integrated() (integrated *http.Request, err error) {
	st.mux.Lock()
	defer st.mux.Unlock()
	if st.arrived == 0 || st.arrived != st.maxIndex {
		return nil, nil
	}
	// Serialize the bodies of all fragments
	bodies := []io.Reader{}
	var contentLength int64
	contentLengthOK := true
	for i := 1; i <= int(st.maxIndex); i++ {
		fragment, ok := st.fragments[i]
		if !ok || fragment == nil {
			return nil, errors.New("missing fragment %d", i)
		}
		if fragment.Body == nil {
			return nil, errors.New("missing body of fragment %d", i)
		}
		bodies = append(bodies, fragment.Body)
		len, err := strconv.ParseInt(fragment.Header.Get("Content-Length"), 10, 64)
		if err != nil {
			contentLengthOK = false
		}
		contentLength += len
	}
	integratedBody := io.MultiReader(bodies...)

	// Set the integrated body on the first fragment
	firstFragment, ok := st.fragments[1]
	if !ok || firstFragment == nil {
		return nil, errors.New("missing first fragment")
	}
	frame.Of(firstFragment).SetFragment(1, 1) // Clear the header
	if contentLengthOK {
		firstFragment.Header.Set("Content-Length", strconv.FormatInt(contentLength, 10))
	}
	firstFragment.Body = io.NopCloser(integratedBody)
	return firstFragment, nil
}

// Add a fragment to be integrated.
// The integrated request is returned if this was the last fragment.
// The fragment count declared by the first fragment is pinned: a later fragment declaring a different count, an
// out-of-range index, or a duplicate index is rejected rather than corrupting the assembly.
func (st *DefragRequest) Add(r *http.Request) (final bool, err error) {
	index, max := frame.Of(r).Fragment()
	if max < 1 || index < 1 || index > max {
		return false, errors.New("invalid fragment %d of %d", index, max, http.StatusBadRequest)
	}
	st.mux.Lock()
	defer st.mux.Unlock()
	if st.maxIndex == 0 {
		st.maxIndex = max
	} else if max != st.maxIndex {
		return false, errors.New("inconsistent fragment count %d, expected %d", max, st.maxIndex, http.StatusBadRequest)
	}
	if _, exists := st.fragments[index]; exists {
		return false, errors.New("duplicate fragment %d", index, http.StatusBadRequest)
	}
	st.fragments[index] = r
	st.arrived++
	st.lastActivity.Store(time.Now().UnixMilli())
	return st.arrived == st.maxIndex, nil
}
