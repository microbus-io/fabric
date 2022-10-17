package connector

import (
	"net/http"
	"time"

	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/frag"
	"github.com/microbus-io/fabric/frame"
)

// defragRequest assembles all fragments of an incoming HTTP request and returns the integrated HTTP request.
// If not all fragments are available yet, it returns nil
func (c *Connector) defragRequest(r *http.Request) (integrated *http.Request, err error) {
	_, fragmentMax := frame.Of(r).Fragment()
	if fragmentMax <= 1 {
		return r, nil
	}
	msgID := frame.Of(r).MessageID()

	c.requestDefragsLock.Lock()
	defragger, ok := c.requestDefrags[msgID]
	if !ok {
		defragger = frag.NewDefragRequest(c.Clock())
		c.requestDefrags[msgID] = defragger
		// Timeout if fragments stop arriving
		go func() {
			for {
				time.Sleep(c.networkHop / 2)
				if defragger.LastActivity() > c.networkHop {
					c.requestDefragsLock.Lock()
					delete(c.requestDefrags, msgID)
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
	delete(c.requestDefrags, msgID)
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
	msgID := frame.Of(r).MessageID()

	c.responseDefragsLock.Lock()
	defragger, ok := c.responseDefrags[msgID]
	if !ok {
		defragger = frag.NewDefragResponse(c.Clock())
		c.responseDefrags[msgID] = defragger
		// Timeout if fragments stop arriving
		go func() {
			for {
				time.Sleep(c.networkHop / 2)
				if defragger.LastActivity() > c.networkHop {
					c.responseDefragsLock.Lock()
					delete(c.responseDefrags, msgID)
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
	delete(c.responseDefrags, msgID)
	c.responseDefragsLock.Unlock()

	return integrated, nil
}
