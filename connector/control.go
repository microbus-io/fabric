package connector

import (
	"net/http"

	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/sub"
)

// subscribeControl creates subscriptions for control requests on the reserved port 888
func (c *Connector) subscribeControl() error {
	// Ping
	err := c.Subscribe(":888/ping", c.handleControlPing, sub.NoQueue())
	if err != nil {
		return errors.Trace(err)
	}
	err = c.Subscribe("https://all:888/ping", c.handleControlPing, sub.NoQueue())
	if err != nil {
		return errors.Trace(err)
	}
	return nil
}

// handleControlPing responds to the :888/ping control request with a pong
func (c *Connector) handleControlPing(w http.ResponseWriter, r *http.Request) error {
	w.Write([]byte("pong"))
	return nil
}
