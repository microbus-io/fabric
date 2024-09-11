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

package middleware

import (
	"net/http"
	"testing"

	"github.com/microbus-io/fabric/connector"
	"github.com/microbus-io/testarossa"
)

func TestChain_CRUD(t *testing.T) {
	noop := func(next connector.HTTPHandler) connector.HTTPHandler {
		return func(w http.ResponseWriter, r *http.Request) error {
			return nil
		}
	}

	chain := &Chain{}
	testarossa.Equal(t, "", chain.String())

	chain.Append("10", noop)
	chain.Append("20", noop)
	testarossa.Equal(t, "10 -> 20", chain.String())
	testarossa.False(t, chain.Exists("5"))
	testarossa.True(t, chain.Exists("10"))
	testarossa.False(t, chain.Exists("15"))
	testarossa.True(t, chain.Exists("20"))

	chain.InsertBefore("10", "5", noop)
	chain.InsertAfter("10", "15", noop)
	testarossa.Equal(t, "5 -> 10 -> 15 -> 20", chain.String())
	testarossa.True(t, chain.Exists("5"))
	testarossa.True(t, chain.Exists("10"))
	testarossa.True(t, chain.Exists("15"))
	testarossa.True(t, chain.Exists("20"))

	chain.Replace("10", noop)
	testarossa.Equal(t, "5 -> 10 -> 15 -> 20", chain.String())

	chain.Delete("10")
	chain.Delete("20")
	testarossa.Equal(t, "5 -> 15", chain.String())

	chain.Prepend("0", noop)
	testarossa.Equal(t, "0 -> 5 -> 15", chain.String())

	chain.Clear()
	testarossa.Equal(t, "", chain.String())

	chain.Replace("10", noop)
	chain.InsertBefore("10", "5", noop)
	chain.InsertAfter("10", "15", noop)
	chain.Delete("20")
	testarossa.Equal(t, "", chain.String())
	testarossa.False(t, chain.Exists("5"))
	testarossa.False(t, chain.Exists("10"))
	testarossa.False(t, chain.Exists("15"))
	testarossa.False(t, chain.Exists("20"))

	chain.Prepend("ALPHA", noop)
	testarossa.Equal(t, "ALPHA", chain.String())
	testarossa.True(t, chain.Exists("ALPHA"))
	testarossa.True(t, chain.Exists("alpha"))
}
