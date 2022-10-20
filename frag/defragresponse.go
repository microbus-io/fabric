package frag

import (
	"io"
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/microbus-io/fabric/clock"
	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/frame"
)

// DefragResponse merges together multiple fragments back into a single HTTP response
type DefragResponse struct {
	fragments    map[int]*http.Response
	maxIndex     int32
	lock         sync.Mutex
	lastActivity time.Time
	clock        clock.Clock
}

// NewDefragResponse creates a new response integrator.
func NewDefragResponse(clock clock.Clock) *DefragResponse {
	return &DefragResponse{
		fragments:    map[int]*http.Response{},
		clock:        clock,
		lastActivity: clock.Now(),
	}
}

// LastActivity indicates how long ago was the last fragment added.
func (st *DefragResponse) LastActivity() time.Duration {
	st.lock.Lock()
	d := st.clock.Since(st.lastActivity)
	st.lock.Unlock()
	return d
}

// Integrated indicates if all the fragments have been collected and if so returns them as a single HTTP response.
func (st *DefragResponse) Integrated() (integrated *http.Response, err error) {
	maxIndex := int(atomic.LoadInt32(&st.maxIndex))
	if maxIndex == 1 {
		return st.fragments[1], nil
	}
	st.lock.Lock()
	defer st.lock.Unlock()

	if maxIndex == 0 || len(st.fragments) != maxIndex {
		return nil, nil
	}

	// Serialize the bodies of all fragments
	bodies := []io.Reader{}
	var contentLength int64
	for i := 1; i <= maxIndex; i++ {
		fragment := st.fragments[i]
		if fragment == nil {
			return nil, errors.Newf("missing fragment %d", i)
		}
		if fragment.Body == nil {
			return nil, errors.Newf("missing body of fragment %d", i)
		}
		bodies = append(bodies, fragment.Body)
		len, err := strconv.ParseInt(fragment.Header.Get("Content-Length"), 10, 64)
		if err != nil {
			return nil, errors.Newf("invalid or missing Content-Length header")
		}
		contentLength += len
	}
	integratedBody := io.MultiReader(bodies...)

	// Set the integrated body on the first fragment
	firstFragment := st.fragments[1]
	if firstFragment == nil {
		return nil, errors.New("missing first fragment")
	}
	frame.Of(firstFragment).SetFragment(1, 1) // Clear the header
	firstFragment.Header.Set("Content-Length", strconv.FormatInt(contentLength, 10))
	firstFragment.Body = io.NopCloser(integratedBody)
	return firstFragment, nil
}

// Add a fragment to be integrated.
func (st *DefragResponse) Add(r *http.Response) error {
	st.lock.Lock()
	index, max := frame.Of(r).Fragment()
	st.fragments[index] = r
	atomic.StoreInt32(&st.maxIndex, int32(max))
	st.lastActivity = st.clock.Now()
	st.lock.Unlock()
	return nil
}
