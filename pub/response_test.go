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
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPub_Response(t *testing.T) {
	t.Parallel()

	myErr := errors.New("my error")
	r := NewErrorResponse(myErr)
	res, err := r.Get()
	assert.Nil(t, res)
	assert.Same(t, myErr, err)

	var myRes http.Response
	r = NewHTTPResponse(&myRes)
	res, err = r.Get()
	assert.NoError(t, err)
	assert.Same(t, &myRes, res)
}
