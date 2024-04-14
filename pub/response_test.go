/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
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
