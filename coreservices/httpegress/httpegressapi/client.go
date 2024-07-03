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

package httpegressapi

import (
	"bytes"
	"context"
	"net/http"

	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/httpx"
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
func (c *Client) Post(ctx context.Context, url string, contentType string, body any) (resp *http.Response, err error) {
	req, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		return nil, errors.Trace(err)
	}
	err = httpx.SetRequestBody(req, body)
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
