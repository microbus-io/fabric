package utils

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestUtils_InfiniteChanTimeout(t *testing.T) {
	t.Parallel()

	n := 4
	inf := NewInfiniteChan[int](n, time.Second)
	assert.Equal(t, n, cap(inf.C))

	// Push twice the capacity of the channel
	t0 := time.Now()
	for i := 0; i < n*2; i++ {
		inf.Push(i)
	}
	t1 := time.Now()
	assert.WithinDuration(t, t0, t1, 50*time.Millisecond, "Channel should not block")
	assert.Len(t, inf.queue, 4)
	assert.Equal(t, n*2, inf.count)

	// Close the channel and wait enough to timeout
	go inf.Close()
	time.Sleep(time.Second + 100*time.Millisecond)

	// Channel should produce its max capacity
	m := 0
	for range inf.C {
		m++
	}
	assert.Equal(t, n, m)
}

func TestUtils_InfiniteChanBeforeTimeout(t *testing.T) {
	t.Parallel()

	n := 4
	inf := NewInfiniteChan[int](n, time.Second)
	assert.Equal(t, n, cap(inf.C))

	// Push twice the capacity of the channel
	t0 := time.Now()
	for i := 0; i < n*2; i++ {
		inf.Push(i)
	}
	t1 := time.Now()
	assert.WithinDuration(t, t0, t1, 50*time.Millisecond, "Channel should not block")
	assert.Len(t, inf.queue, 4)
	assert.Equal(t, n*2, inf.count)

	// Close the channel
	go inf.Close()

	// Channel should produce all elements if read quickly after closing
	m := 0
	for range inf.C {
		m++
		time.Sleep(25 * time.Millisecond)
	}
	assert.Equal(t, n*2, m)
}
