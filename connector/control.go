package connector

import (
	"net/http"

	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/sub"
)

// subscribeControl creates subscriptions for control requests on the reserved port 888.
func (c *Connector) subscribeControl() error {
	type ctrlSub struct {
		path    string
		handler HTTPHandler
		options []sub.Option
	}
	subs := []*ctrlSub{
		{
			path:    "ping",
			handler: c.handleControlPing,
			options: []sub.Option{sub.NoQueue()},
		},
		{
			path:    "config-refresh",
			handler: c.handleControlConfigRefresh,
			options: []sub.Option{sub.NoQueue()},
		},
	}
	for _, s := range subs {
		err := c.Subscribe(":888/"+s.path, s.handler, s.options...)
		if err != nil {
			return errors.Trace(err)
		}
		err = c.Subscribe("https://all:888/"+s.path, s.handler, s.options...)
		if err != nil {
			return errors.Trace(err)
		}
	}
	return nil
}

// handleControlPing responds to the :888/ping control request with a pong.
func (c *Connector) handleControlPing(w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"pong":0}`))
	return nil
}

// handleControlConfigRefresh responds to the :888/config-refresh control request
// by pulling the latest config values from the configurator service.
func (c *Connector) handleControlConfigRefresh(w http.ResponseWriter, r *http.Request) error {
	err := c.refreshConfig(r.Context())
	if err != nil {
		return errors.Trace(err)
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte("{}"))
	return nil
}
