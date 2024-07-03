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

package connector

import (
	"net/http"
	"time"

	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/frame"
	"github.com/microbus-io/fabric/httpx"
)

// defragRequest assembles all fragments of an incoming HTTP request and returns the integrated HTTP request.
// If not all fragments are available yet, it returns nil
func (c *Connector) defragRequest(r *http.Request) (integrated *http.Request, err error) {
	fragmentIndex, fragmentMax := frame.Of(r).Fragment()
	if fragmentMax <= 1 {
		return r, nil
	}
	fromID := frame.Of(r).FromID()
	msgID := frame.Of(r).MessageID()
	fragKey := fromID + "|" + msgID

	defragger, ok := c.requestDefrags.Load(fragKey)
	if !ok {
		if fragmentIndex != 1 {
			// Most likely caused after a timeout, but can also happen if initial chunk has wrong index
			return nil, errors.Newc(http.StatusRequestTimeout, "defrag timeout")
		}
		defragger = httpx.NewDefragRequest()
		c.requestDefrags.Store(fragKey, defragger)
		// Timeout if fragments stop arriving
		go func() {
			for {
				time.Sleep(c.networkHop / 2)
				if defragger.LastActivity() > c.networkHop {
					c.requestDefrags.Delete(fragKey)
					break
				}
			}
		}()
	}

	err = defragger.Add(r)
	if err != nil {
		return nil, errors.Trace(err)
	}
	integrated, err = defragger.Integrated()
	if err != nil {
		return nil, errors.Trace(err)
	}
	if integrated == nil {
		// Not all fragments arrived yet
		return nil, nil
	}

	c.requestDefrags.Delete(fragKey)

	return integrated, nil
}

// defragResponse assembles all fragments of an incoming HTTP response and returns the integrated HTTP request.
// If not all fragments are available yet, it returns nil
func (c *Connector) defragResponse(r *http.Response) (integrated *http.Response, err error) {
	fragmentIndex, fragmentMax := frame.Of(r).Fragment()
	if fragmentMax <= 1 {
		return r, nil
	}
	fromID := frame.Of(r).FromID()
	msgID := frame.Of(r).MessageID()
	fragKey := fromID + "|" + msgID

	defragger, ok := c.responseDefrags.Load(fragKey)
	if !ok {
		if fragmentIndex != 1 {
			// Most likely caused after a timeout, but can also happen if initial chunk has wrong index
			return nil, errors.Newc(http.StatusRequestTimeout, "defrag timeout")
		}
		defragger = httpx.NewDefragResponse()
		c.responseDefrags.Store(fragKey, defragger)
		// Timeout if fragments stop arriving
		go func() {
			for {
				time.Sleep(c.networkHop / 2)
				if defragger.LastActivity() > c.networkHop {
					c.responseDefrags.Delete(fragKey)
					break
				}
			}
		}()
	}

	err = defragger.Add(r)
	if err != nil {
		return nil, errors.Trace(err)
	}
	integrated, err = defragger.Integrated()
	if err != nil {
		return nil, errors.Trace(err)
	}
	if integrated == nil {
		// Not all fragments arrived yet
		return nil, nil
	}

	c.responseDefrags.Delete(fragKey)

	return integrated, nil
}
