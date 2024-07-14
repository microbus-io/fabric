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

package frame

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/microbus-io/testarossa"
)

func TestFrame_Of(t *testing.T) {
	t.Parallel()

	// http.Request
	httpRequest, err := http.NewRequest("GET", "https://www.example.com", nil)
	testarossa.NoError(t, err)
	httpRequest.Header.Set(HeaderMsgId, "123")
	testarossa.Equal(t, "123", Of(httpRequest).MessageID())

	// httptest.ResponseRecorder and http.Response
	httpRecorder := httptest.NewRecorder()
	httpRecorder.Header().Set(HeaderMsgId, "123")
	testarossa.Equal(t, "123", Of(httpRecorder).MessageID())
	httpResponse := httpRecorder.Result()
	testarossa.Equal(t, "123", Of(httpResponse).MessageID())

	// http.Header
	hdr := make(http.Header)
	hdr.Set(HeaderMsgId, "123")
	testarossa.Equal(t, "123", Of(hdr).MessageID())

	// context.Context
	ctx := context.WithValue(context.Background(), contextKey, hdr)
	testarossa.Equal(t, "123", Of(ctx).MessageID())

	// Empty context.Context should not panic
	testarossa.Equal(t, "", Of(context.Background()).MessageID())
}

func TestFrame_GetSet(t *testing.T) {
	t.Parallel()

	f := Of(make(http.Header))

	testarossa.Equal(t, "", f.OpCode())
	f.SetOpCode(OpCodeError)
	testarossa.Equal(t, OpCodeError, f.OpCode())
	f.SetOpCode("")
	testarossa.Equal(t, "", f.OpCode())

	testarossa.Zero(t, f.CallDepth())
	f.SetCallDepth(123)
	testarossa.Equal(t, 123, f.CallDepth())
	f.SetCallDepth(0)
	testarossa.Zero(t, f.CallDepth())

	testarossa.Equal(t, "", f.FromHost())
	f.SetFromHost("www.example.com")
	testarossa.Equal(t, "www.example.com", f.FromHost())
	f.SetFromHost("")
	testarossa.Equal(t, "", f.FromHost())

	testarossa.Equal(t, "", f.FromID())
	f.SetFromID("1234567890")
	testarossa.Equal(t, "1234567890", f.FromID())
	f.SetFromID("")
	testarossa.Equal(t, "", f.FromID())

	testarossa.Zero(t, f.FromVersion())
	f.SetFromVersion(12345)
	testarossa.Equal(t, 12345, f.FromVersion())
	f.SetFromVersion(0)
	testarossa.Zero(t, f.FromVersion())

	testarossa.Equal(t, "", f.MessageID())
	f.SetMessageID("1234567890")
	testarossa.Equal(t, "1234567890", f.MessageID())
	f.SetMessageID("")
	testarossa.Equal(t, "", f.MessageID())

	budget := f.TimeBudget()
	testarossa.Equal(t, time.Duration(0), budget)
	f.SetTimeBudget(123 * time.Second)
	budget = f.TimeBudget()
	testarossa.Equal(t, 123*time.Second, budget)
	f.SetTimeBudget(0)
	budget = f.TimeBudget()
	testarossa.Equal(t, time.Duration(0), budget)

	testarossa.Equal(t, "", f.Queue())
	f.SetQueue("1234567890")
	testarossa.Equal(t, "1234567890", f.Queue())
	f.SetQueue("")
	testarossa.Equal(t, "", f.Queue())

	fi, fm := f.Fragment()
	testarossa.Equal(t, 1, fi)
	testarossa.Equal(t, 1, fm)
	f.SetFragment(2, 5)
	fi, fm = f.Fragment()
	testarossa.Equal(t, fi, 2)
	testarossa.Equal(t, fm, 5)
	f.SetFragment(0, 0)
	fi, fm = f.Fragment()
	testarossa.Equal(t, fi, 1)
	testarossa.Equal(t, fm, 1)
}

func TestFrame_XForwarded(t *testing.T) {
	httpRequest, err := http.NewRequest("GET", "https://www.example.com", nil)
	testarossa.NoError(t, err)
	frame := Of(httpRequest)
	testarossa.Equal(t, "", frame.XForwardedBaseURL())

	httpRequest.Header.Set("X-Forwarded-Proto", "https")
	httpRequest.Header.Set("X-Forwarded-Host", "www.proxy.com")
	httpRequest.Header.Set("X-Forwarded-Prefix", "/example")
	testarossa.Equal(t, "https://www.proxy.com/example", frame.XForwardedBaseURL())

	httpRequest.Header.Set("X-Forwarded-Prefix", "/example/")
	testarossa.Equal(t, "https://www.proxy.com/example", frame.XForwardedBaseURL())
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
		testarossa.SliceEqual(t, expected, langs)
	}
}
