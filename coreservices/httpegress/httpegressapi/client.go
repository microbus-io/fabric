/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package httpegressapi

import (
	"bytes"
	"context"
	"io"
	"net/http"

	"github.com/microbus-io/fabric/errors"
)

// Get makes a GET request to a URL, respecting the timeout set in the context.
func (c *Client) Get(ctx context.Context, url string) (resp *http.Response, err error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, errors.Trace(err)
	}
	var buf bytes.Buffer
	err = req.WriteProxy(&buf)
	if err != nil {
		return nil, errors.Trace(err)
	}
	resp, err = c.MakeRequest(ctx, "", "", &buf)
	return resp, errors.Trace(err)
}

// Post makes a POST request to a URL, respecting the timeout set in the context.
func (c *Client) Post(ctx context.Context, url string, contentType string, body io.Reader) (resp *http.Response, err error) {
	req, err := http.NewRequest(http.MethodPost, url, body)
	if err != nil {
		return nil, errors.Trace(err)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	var buf bytes.Buffer
	err = req.WriteProxy(&buf)
	if err != nil {
		return nil, errors.Trace(err)
	}
	resp, err = c.MakeRequest(ctx, "", "", &buf)
	return resp, errors.Trace(err)
}

// Do makes a request to a URL, respecting the timeout set in the context.
func (c *Client) Do(ctx context.Context, req *http.Request) (resp *http.Response, err error) {
	var buf bytes.Buffer
	err = req.WriteProxy(&buf)
	if err != nil {
		return nil, errors.Trace(err)
	}
	resp, err = c.MakeRequest(ctx, "", "", &buf)
	return resp, errors.Trace(err)
}
