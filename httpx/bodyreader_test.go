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

package httpx

import (
	"io"
	"testing"

	"github.com/microbus-io/testarossa"
)

func TestHttpx_BodyReader(t *testing.T) {
	bin := []byte("Lorem Ipsum")
	br := NewBodyReader(bin)
	bout, err := io.ReadAll(br)
	testarossa.NoError(t, err)
	testarossa.SliceEqual(t, bin, bout)
	testarossa.SliceEqual(t, bin, br.Bytes())
	br.Reset()
	bout, err = io.ReadAll(br)
	testarossa.NoError(t, err)
	testarossa.SliceEqual(t, bin, bout)
	testarossa.SliceEqual(t, bin, br.Bytes())
	err = br.Close()
	testarossa.NoError(t, err)
}
