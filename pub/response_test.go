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
