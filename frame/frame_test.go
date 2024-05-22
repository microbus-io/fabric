/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package frame

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFrame_Of(t *testing.T) {
	t.Parallel()

	// http.Request
	httpRequest, err := http.NewRequest("GET", "https://www.example.com", nil)
	assert.NoError(t, err)
	httpRequest.Header.Set(HeaderMsgId, "123")
	assert.Equal(t, "123", Of(httpRequest).MessageID())

	// httptest.ResponseRecorder and http.Response
	httpRecorder := httptest.NewRecorder()
	httpRecorder.Header().Set(HeaderMsgId, "123")
	assert.Equal(t, "123", Of(httpRecorder).MessageID())
	httpResponse := httpRecorder.Result()
	assert.Equal(t, "123", Of(httpResponse).MessageID())

	// http.Header
	hdr := make(http.Header)
	hdr.Set(HeaderMsgId, "123")
	assert.Equal(t, "123", Of(hdr).MessageID())

	// context.Context
	ctx := context.WithValue(context.Background(), contextKey, hdr)
	assert.Equal(t, "123", Of(ctx).MessageID())

	// Empty context.Context should not panic
	assert.Equal(t, "", Of(context.Background()).MessageID())
}

func TestFrame_GetSet(t *testing.T) {
	t.Parallel()

	f := Of(make(http.Header))

	assert.Equal(t, "", f.OpCode())
	f.SetOpCode(OpCodeError)
	assert.Equal(t, OpCodeError, f.OpCode())
	f.SetOpCode("")
	assert.Equal(t, "", f.OpCode())

	assert.Equal(t, 0, f.CallDepth())
	f.SetCallDepth(123)
	assert.Equal(t, 123, f.CallDepth())
	f.SetCallDepth(0)
	assert.Equal(t, 0, f.CallDepth())

	assert.Equal(t, "", f.FromHost())
	f.SetFromHost("www.example.com")
	assert.Equal(t, "www.example.com", f.FromHost())
	f.SetFromHost("")
	assert.Equal(t, "", f.FromHost())

	assert.Equal(t, "", f.FromID())
	f.SetFromID("1234567890")
	assert.Equal(t, "1234567890", f.FromID())
	f.SetFromID("")
	assert.Equal(t, "", f.FromID())

	assert.Equal(t, 0, f.FromVersion())
	f.SetFromVersion(12345)
	assert.Equal(t, 12345, f.FromVersion())
	f.SetFromVersion(0)
	assert.Equal(t, 0, f.FromVersion())

	assert.Equal(t, "", f.MessageID())
	f.SetMessageID("1234567890")
	assert.Equal(t, "1234567890", f.MessageID())
	f.SetMessageID("")
	assert.Equal(t, "", f.MessageID())

	budget := f.TimeBudget()
	assert.Equal(t, time.Duration(0), budget)
	f.SetTimeBudget(123 * time.Second)
	budget = f.TimeBudget()
	assert.Equal(t, 123*time.Second, budget)
	f.SetTimeBudget(0)
	budget = f.TimeBudget()
	assert.Equal(t, time.Duration(0), budget)

	assert.Equal(t, "", f.Queue())
	f.SetQueue("1234567890")
	assert.Equal(t, "1234567890", f.Queue())
	f.SetQueue("")
	assert.Equal(t, "", f.Queue())

	fi, fm := f.Fragment()
	assert.Equal(t, 1, fi)
	assert.Equal(t, 1, fm)
	f.SetFragment(2, 5)
	fi, fm = f.Fragment()
	assert.Equal(t, fi, 2)
	assert.Equal(t, fm, 5)
	f.SetFragment(0, 0)
	fi, fm = f.Fragment()
	assert.Equal(t, fi, 1)
	assert.Equal(t, fm, 1)
}

func TestFrame_XForwarded(t *testing.T) {
	httpRequest, err := http.NewRequest("GET", "https://www.example.com", nil)
	assert.NoError(t, err)
	frame := Of(httpRequest)
	assert.Equal(t, "", frame.XForwardedBaseURL())

	httpRequest.Header.Set("X-Forwarded-Proto", "https")
	httpRequest.Header.Set("X-Forwarded-Host", "www.proxy.com")
	httpRequest.Header.Set("X-Forwarded-Prefix", "/example")
	assert.Equal(t, "https://www.proxy.com/example", frame.XForwardedBaseURL())

	httpRequest.Header.Set("X-Forwarded-Prefix", "/example/")
	assert.Equal(t, "https://www.proxy.com/example", frame.XForwardedBaseURL())
}

func TestFrame_Languages(t *testing.T) {
	testCases := []string{
		"", "",
		"en", "en",
		"EN", "EN",
		"da, en-gb;q=0.8, en;q=0.7", "da,en-gb,en",
		"da, en-gb;q=0.7, en;q=0.8", "da,en,en-gb",
		"da,en-gb;q=0.7,en;q=0.8", "da,en,en-gb",
		" en ;q=1   , es ; q = 0.5 ", "en,es",
	}
	h := http.Header{}
	for i := 0; i < len(testCases); i += 2 {
		h.Set("Accept-Language", testCases[i])
		langs := Of(h).Languages()
		var expected []string
		if testCases[i+1] != "" {
			expected = strings.Split(testCases[i+1], ",")
		}
		assert.Equal(t, expected, langs)
	}
}
