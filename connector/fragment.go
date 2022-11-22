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
	_, fragmentMax := frame.Of(r).Fragment()
	if fragmentMax <= 1 {
		return r, nil
	}
	fromID := frame.Of(r).FromID()
	msgID := frame.Of(r).MessageID()
	fragKey := fromID + "|" + msgID

	c.requestDefragsLock.Lock()
	defragger, ok := c.requestDefrags[fragKey]
	if !ok {
		defragger = httpx.NewDefragRequest()
		c.requestDefrags[fragKey] = defragger
		// Timeout if fragments stop arriving
		go func() {
			for {
				time.Sleep(c.networkHop / 2)
				if defragger.LastActivity() > c.networkHop {
					c.requestDefragsLock.Lock()
					delete(c.requestDefrags, fragKey)
					c.requestDefragsLock.Unlock()
					break
				}
			}
		}()
	}
	c.requestDefragsLock.Unlock()

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

	c.requestDefragsLock.Lock()
	delete(c.requestDefrags, fragKey)
	c.requestDefragsLock.Unlock()

	return integrated, nil
}

// defragResponse assembles all fragments of an incoming HTTP response and returns the integrated HTTP request.
// If not all fragments are available yet, it returns nil
func (c *Connector) defragResponse(r *http.Response) (integrated *http.Response, err error) {
	_, fragmentMax := frame.Of(r).Fragment()
	if fragmentMax <= 1 {
		return r, nil
	}
	fromID := frame.Of(r).FromID()
	msgID := frame.Of(r).MessageID()
	fragKey := fromID + "|" + msgID

	c.responseDefragsLock.Lock()
	defragger, ok := c.responseDefrags[fragKey]
	if !ok {
		defragger = httpx.NewDefragResponse()
		c.responseDefrags[fragKey] = defragger
		// Timeout if fragments stop arriving
		go func() {
			for {
				time.Sleep(c.networkHop / 2)
				if defragger.LastActivity() > c.networkHop {
					c.responseDefragsLock.Lock()
					delete(c.responseDefrags, fragKey)
					c.responseDefragsLock.Unlock()
					break
				}
			}
		}()
	}
	c.responseDefragsLock.Unlock()

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

	c.responseDefragsLock.Lock()
	delete(c.responseDefrags, fragKey)
	c.responseDefragsLock.Unlock()

	return integrated, nil
}
