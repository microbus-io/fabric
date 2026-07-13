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

package connector

import (
	"net/http"
	"time"

	"github.com/microbus-io/errors"
	"github.com/microbus-io/fabric/frame"
	"github.com/microbus-io/fabric/httpx"
)

// admitFirstFragment creates the reassembly entry for a multi-fragment request, carrying the lifetime deadline.
// It returns false if an entry for the key already exists (a duplicate first fragment).
func (c *Connector) admitFirstFragment(r *http.Request) (admitted bool) {
	fromID := frame.Of(r).FromID()
	msgID := frame.Of(r).MessageID()
	key := fromID + "|" + msgID
	d := httpx.NewDefragRequest()
	deadline := time.Now().Add(frame.Of(r).TimeBudget())
	return c.requestDefrags.admit(key, d, deadline)
}

// defragRequest assembles the fragments of an incoming request. It returns the integrated request once the last
// fragment arrives, or a nil request for a non-final fragment.
func (c *Connector) defragRequest(r *http.Request) (integrated *http.Request, err error) {
	_, fragmentMax := frame.Of(r).Fragment()
	if fragmentMax <= 1 {
		return r, nil
	}
	fromID := frame.Of(r).FromID()
	msgID := frame.Of(r).MessageID()
	key := fromID + "|" + msgID

	// Every fragment - including the first - joins the entry created synchronously at fragment 1.
	d, ok := c.requestDefrags.defragger(key)
	if !ok {
		// Reject if no live entry (never admitted, or already swept)
		return nil, errors.New("unknown or expired transfer", http.StatusRequestTimeout)
	}
	integrated, err = c.addRequestFragment(key, d, r)
	return integrated, errors.Trace(err)
}

// addRequestFragment adds one fragment to an admitted request transfer, returning the integrated request on the
// final fragment, or a nil request while fragments are still outstanding. A framing violation drops the transfer.
func (c *Connector) addRequestFragment(key string, d *httpx.DefragRequest, r *http.Request) (integrated *http.Request, err error) {
	final, err := d.Add(r)
	if err != nil {
		c.requestDefrags.remove(key)
		return nil, errors.Trace(err)
	}
	if !final {
		return nil, nil
	}
	c.requestDefrags.remove(key)
	integrated, err = d.Integrated()
	return integrated, errors.Trace(err)
}

// defragResponse assembles the fragments of an incoming response, mirroring defragRequest.
func (c *Connector) defragResponse(r *http.Response) (integrated *http.Response, err error) {
	index, fragmentMax := frame.Of(r).Fragment()
	if fragmentMax <= 1 {
		return r, nil
	}
	fromID := frame.Of(r).FromID()
	msgID := frame.Of(r).MessageID()
	key := fromID + "|" + msgID

	if index == 1 {
		d := httpx.NewDefragResponse()
		deadline := time.Now().Add(c.maxTimeBudget)
		if !c.responseDefrags.admit(key, d, deadline) {
			return nil, nil
		}
		return c.addResponseFragment(key, d, r)
	}

	d, ok := c.responseDefrags.defragger(key)
	if !ok {
		return nil, errors.New("unknown or expired transfer", http.StatusRequestTimeout)
	}
	integrated, err = c.addResponseFragment(key, d, r)
	return integrated, errors.Trace(err)
}

// addResponseFragment adds one fragment to an admitted response transfer, mirroring addRequestFragment.
func (c *Connector) addResponseFragment(key string, d *httpx.DefragResponse, r *http.Response) (integrated *http.Response, err error) {
	final, err := d.Add(r)
	if err != nil {
		c.responseDefrags.remove(key)
		return nil, errors.Trace(err)
	}
	if !final {
		return nil, nil
	}
	c.responseDefrags.remove(key)
	integrated, err = d.Integrated()
	return integrated, errors.Trace(err)
}
