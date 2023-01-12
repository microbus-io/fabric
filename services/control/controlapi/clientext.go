/*
Package controlapi implements the public API of the control.sys microservice,
including clients and data structures.

This microservice is created for the sake of generating the client API for the :888 control subscriptions.
The microservice itself does nothing and should not be included in applications.
*/
package controlapi

import (
	"context"

	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/frame"
	"github.com/microbus-io/fabric/pub"
)

// Pong is returned as a response to a ping.
type Pong struct {
	Service string
}

// PingPongResponse holds a single response of a call to Ping.
// Use the Get method to obtain the return values.
type PingPongResponse struct {
	pong *Pong
	err  error
}

// Get obtains the return values of the call to TokenValidate.
func (_r *PingPongResponse) Get() (pong *Pong, err error) {
	return _r.pong, _r.err
}

// PingServices performs a ping and dedups the result on a per-service basis.
func (_c *Client) PingServices(ctx context.Context, options ...pub.Option) <-chan *PingPongResponse {
	ch := NewMulticastClient(_c.svc).ForHost(_c.host).Ping(ctx, options...)
	filtered := make(chan *PingPongResponse, cap(ch))
	go func() {
		seen := map[string]bool{}
		for pingRes := range ch {
			if pingRes.err != nil {
				pingRes.err = errors.Trace(pingRes.err)
				filtered <- &PingPongResponse{err: errors.Trace(pingRes.err)}
				continue
			}
			from := frame.Of(pingRes.HTTPResponse).FromHost()
			if seen[from] {
				continue
			}
			seen[from] = true
			filtered <- &PingPongResponse{pong: &Pong{
				Service: from,
			}}
		}
		close(filtered)
	}()
	return filtered
}
