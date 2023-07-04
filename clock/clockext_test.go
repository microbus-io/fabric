/*
Copyright (c) 2023 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package clock

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestClock_NewMockAt(t *testing.T) {
	t.Parallel()

	tm := time.Date(2022, 10, 27, 20, 26, 15, 0, time.Local)
	mock := NewMockAt(tm)
	assert.True(t, tm.Equal(mock.Now()))

	mock = NewMockAtDate(2022, 10, 27, 20, 26, 15, 0, time.Local)
	assert.True(t, tm.Equal(mock.Now()))

	mock = NewMockAtNow()
	assert.WithinDuration(t, time.Now(), mock.Now(), 100*time.Millisecond)
}
