package connector

import (
	"net/http"

	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/sub"
)

// subscribeControl creates subscriptions for control requests on the reserved port 888
func (c *Connector) subscribeControl() error {
	type ctrlSub struct {
		path    string
		f       sub.HTTPHandler
		options []sub.Option
	}
	subs := []*ctrlSub{
		{
			path:    "ping",
			f:       c.handleControlPing,
			options: []sub.Option{sub.NoQueue()},
		},
		{
			path:    "config/refresh",
			f:       c.handleControlConfigRefresh,
			options: []sub.Option{sub.NoQueue()},
		},
	}
	for _, sub := range subs {
		err := c.Subscribe(":888/"+sub.path, sub.f, sub.options...)
		if err != nil {
			return errors.Trace(err)
		}
		err = c.Subscribe("https://all:888/"+sub.path, sub.f, sub.options...)
		if err != nil {
			return errors.Trace(err)
		}
	}
	return nil
}

// handleControlPing responds to the :888/ping control request with a pong
func (c *Connector) handleControlPing(w http.ResponseWriter, r *http.Request) error {
	w.Write([]byte(`pong`))
	return nil
}

// handleControlPing responds to the :888/ping control request with a pong
func (c *Connector) handleControlConfigRefresh(w http.ResponseWriter, r *http.Request) error {
	err := c.refreshConfig(r.Context())
	if err != nil {
		return errors.Trace(err)
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte("{}"))
	return nil
}
