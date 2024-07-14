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

package utils

import (
	"encoding/hex"
	"testing"

	"github.com/microbus-io/testarossa"
)

func TestUtils_SourceCodeHash(t *testing.T) {
	t.Parallel()

	h, err := SourceCodeSHA256(".")
	testarossa.NoError(t, err)
	b, err := hex.DecodeString(h)
	testarossa.NoError(t, err)
	testarossa.SliceLen(t, b, 256/8)
}
