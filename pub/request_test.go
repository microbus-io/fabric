/*
Copyright 2023 Microbus LLC and various contributors

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

package pub

import (
	"bytes"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/frame"
	"github.com/stretchr/testify/assert"
)

func TestPub_MethodAndURL(t *testing.T) {
	t.Parallel()

	// GET
	req, err := NewRequest([]Option{
		GET("https://www.example.com"),
	}...)
	assert.NoError(t, err)
	httpReq, err := toHTTP(req)
	assert.NoError(t, err)
	assert.Equal(t, "GET", httpReq.Method)
	assert.Equal(t, "www.example.com", httpReq.URL.Hostname())

	// POST
	req, err = NewRequest([]Option{
		POST("https://www.example.com/path"),
	}...)
	assert.NoError(t, err)
	httpReq, err = toHTTP(req)
	assert.NoError(t, err)
	assert.Equal(t, "POST", httpReq.Method)
	assert.Equal(t, "www.example.com", httpReq.URL.Hostname())
	assert.Equal(t, "/path", httpReq.URL.Path)

	// Any method
	req, err = NewRequest([]Option{
		Method("Delete"), // Mixed case
		URL("https://www.example.com"),
	}...)
	assert.NoError(t, err)
	httpReq, err = toHTTP(req)
	assert.NoError(t, err)
	assert.Equal(t, "DELETE", httpReq.Method)
	assert.Equal(t, "www.example.com", httpReq.URL.Hostname())
}

func TestPub_Header(t *testing.T) {
	t.Parallel()

	req, err := NewRequest([]Option{
		GET("https://www.example.com"),
		Header("Content-Type", "text/html"),
		Header("X-SOMETHING", "Else"), // Uppercase
	}...)
	assert.NoError(t, err)
	httpReq, err := toHTTP(req)
	assert.NoError(t, err)
	assert.Equal(t, "text/html", httpReq.Header.Get("Content-Type"))
	assert.Equal(t, "Else", httpReq.Header.Get("X-Something"))
}

func TestPub_Body(t *testing.T) {
	t.Parallel()

	// String
	req, err := NewRequest([]Option{
		GET("https://www.example.com"),
		Body("Hello World"),
	}...)
	assert.NoError(t, err)
	httpReq, err := toHTTP(req)
	assert.NoError(t, err)
	body, err := io.ReadAll(httpReq.Body)
	assert.NoError(t, err)
	assert.Equal(t, "Hello World", string(body))

	// []byte
	req, err = NewRequest([]Option{
		GET("https://www.example.com"),
		Body([]byte("Hello World")),
	}...)
	assert.NoError(t, err)
	httpReq, err = toHTTP(req)
	assert.NoError(t, err)
	body, err = io.ReadAll(httpReq.Body)
	assert.NoError(t, err)
	assert.Equal(t, "Hello World", string(body))

	// io.Reader
	req, err = NewRequest([]Option{
		GET("https://www.example.com"),
		Body(bytes.NewReader([]byte("Hello World"))),
	}...)
	assert.NoError(t, err)
	httpReq, err = toHTTP(req)
	assert.NoError(t, err)
	body, err = io.ReadAll(httpReq.Body)
	assert.NoError(t, err)
	assert.Equal(t, "Hello World", string(body))

	// JSON
	j := struct {
		S string `json:"s"`
		I int    `json:"i"`
	}{"ABC", 123}
	req, err = NewRequest([]Option{
		GET("https://www.example.com"),
		Body(j),
	}...)
	assert.NoError(t, err)
	httpReq, err = toHTTP(req)
	assert.NoError(t, err)
	body, err = io.ReadAll(httpReq.Body)
	assert.NoError(t, err)
	assert.Equal(t, `{"s":"ABC","i":123}`, string(body))
}

func TestPub_TimeBudget(t *testing.T) {
	t.Parallel()

	req, err := NewRequest([]Option{
		GET("https://www.example.com"),
		TimeBudget(30 * time.Second),
		TimeBudget(20 * time.Second),
		TimeBudget(40 * time.Second), // Last
	}...)
	assert.NoError(t, err)
	httpReq, err := toHTTP(req)
	assert.NoError(t, err)
	budget := frame.Of(httpReq).TimeBudget()
	assert.Equal(t, 40*time.Second, budget)
}

func toHTTP(req *Request) (*http.Request, error) {
	httpReq, err := http.NewRequest(req.Method, req.URL, req.Body)
	if err != nil {
		return nil, errors.Trace(err)
	}
	for name, value := range req.Header {
		httpReq.Header[name] = value
	}
	frame.Of(httpReq).SetTimeBudget(req.TimeBudget)
	return httpReq, nil
}

func TestPub_Canonical(t *testing.T) {
	t.Parallel()

	r, err := NewRequest(GET("https://www.example.com:567/path?a=5&b=6")) // https
	assert.NoError(t, err)
	assert.Equal(t, "https://www.example.com:567/path", r.Canonical())

	r, err = NewRequest(GET("http://www.example.com/path")) // http
	assert.NoError(t, err)
	assert.Equal(t, "http://www.example.com:80/path", r.Canonical())

	r, err = NewRequest(GET("//www.example.com/path")) // no scheme
	assert.NoError(t, err)
	assert.Equal(t, "https://www.example.com:443/path", r.Canonical())
}

func TestPub_Apply(t *testing.T) {
	t.Parallel()

	r, err := NewRequest()
	assert.NoError(t, err)

	r.Apply(URL("https://www.example.com/delete"), Method("DELETE"))
	assert.Equal(t, "DELETE", r.Method)
	assert.Equal(t, "https://www.example.com:443/delete", r.Canonical())

	r.Apply(GET("https://www.example.com/get"))
	assert.Equal(t, "GET", r.Method)
	assert.Equal(t, "https://www.example.com:443/get", r.Canonical())

	r.Apply(POST("https://www.example.com/post"))
	assert.Equal(t, "POST", r.Method)
	assert.Equal(t, "https://www.example.com:443/post", r.Canonical())

	r.Apply(Multicast())
	assert.Equal(t, true, r.Multicast)

	r.Apply(Unicast())
	assert.Equal(t, false, r.Multicast)

	r.Apply(Body("lorem ipsum"))
	body, err := io.ReadAll(r.Body)
	assert.NoError(t, err)
	assert.Equal(t, "lorem ipsum", string(body))

	r.Apply(Header("Foo", "Bar"))
	assert.Equal(t, "Bar", r.Header.Get("Foo"))
}

func TestPub_QueryArgs(t *testing.T) {
	t.Parallel()

	req, err := NewRequest(GET("https://www.example.com:443/path?a=1"))
	assert.NoError(t, err)
	httpReq, err := toHTTP(req)
	assert.NoError(t, err)
	assert.Equal(t, "https://www.example.com:443/path?a=1", httpReq.URL.String())

	err = req.Apply(QueryArg("b", "2"))
	assert.NoError(t, err)
	httpReq, err = toHTTP(req)
	assert.NoError(t, err)
	assert.Equal(t, "https://www.example.com:443/path?a=1&b=2", httpReq.URL.String())

	err = req.Apply(QueryArg("a", "3"))
	assert.NoError(t, err)
	httpReq, err = toHTTP(req)
	assert.NoError(t, err)
	assert.Equal(t, "https://www.example.com:443/path?a=1&b=2&a=3", httpReq.URL.String())

	err = req.Apply(URL("https://zzz.example.com:123/newpath"))
	assert.NoError(t, err)
	httpReq, err = toHTTP(req)
	assert.NoError(t, err)
	assert.Equal(t, "https://zzz.example.com:123/newpath?b=2&a=3", httpReq.URL.String())

	err = req.Apply(Query("m=5&n=6"))
	assert.NoError(t, err)
	httpReq, err = toHTTP(req)
	assert.NoError(t, err)
	assert.Equal(t, "https://zzz.example.com:123/newpath?b=2&a=3&m=5&n=6", httpReq.URL.String())
}
