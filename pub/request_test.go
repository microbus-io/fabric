package pub

import (
	"bytes"
	"io"
	"testing"
	"time"

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
	httpReq, err := req.ToHTTP()
	assert.NoError(t, err)
	assert.Equal(t, "GET", httpReq.Method)
	assert.Equal(t, "www.example.com", httpReq.URL.Hostname())

	// POST
	req, err = NewRequest([]Option{
		POST("https://www.example.com/path"),
	}...)
	assert.NoError(t, err)
	httpReq, err = req.ToHTTP()
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
	httpReq, err = req.ToHTTP()
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
	httpReq, err := req.ToHTTP()
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
	httpReq, err := req.ToHTTP()
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
	httpReq, err = req.ToHTTP()
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
	httpReq, err = req.ToHTTP()
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
	httpReq, err = req.ToHTTP()
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
		TimeBudget(20 * time.Second), // Shortest
		TimeBudget(40 * time.Second),
	}...)
	assert.NoError(t, err)
	httpReq, err := req.ToHTTP()
	assert.NoError(t, err)
	budget := frame.Of(httpReq).TimeBudget()
	assert.True(t, budget <= 20*time.Second && budget >= 19*time.Second)
}
