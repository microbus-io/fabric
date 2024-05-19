/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package utils

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestUtils_InfiniteChanSlowReadAfterClose(t *testing.T) {
	t.Parallel()

	n := 4
	inf := MakeInfiniteChan[int](n)
	assert.Equal(t, n, cap(inf.C()))

	// Push twice the capacity of the channel
	t0 := time.Now()
	for i := 0; i < n*2; i++ {
		inf.Push(i)
	}
	t1 := time.Now()
	assert.WithinDuration(t, t0, t1, 50*time.Millisecond, "Channel should not block")
	assert.Len(t, inf.queue, 4)
	assert.Equal(t, int32(n*2), inf.pushed.Load())

	// Close the channel and wait enough to timeout
	go inf.Close(time.Second)
	time.Sleep(time.Second + 100*time.Millisecond)

	// Channel should produce its max capacity
	m := 0
	for range inf.C() {
		m++
	}
	assert.Equal(t, n, m)
}

func TestUtils_InfiniteChanClosed(t *testing.T) {
	t.Parallel()

	// Under capacity
	inf := MakeInfiniteChan[int](1)
	inf.Push(1)
	<-inf.C()
	inf.Close(time.Millisecond * 100)
	assert.True(t, inf.closed.Load())
	assert.Panics(t, func() { inf.Push(999) })
	assert.Panics(t, func() { inf.ch <- 999 })

	// Over capacity
	inf = MakeInfiniteChan[int](1)
	inf.Push(1)
	inf.Push(1)
	<-inf.C()
	<-inf.C()
	inf.Close(time.Millisecond * 100)
	assert.True(t, inf.closed.Load())
	assert.Panics(t, func() { inf.Push(999) })
	assert.Panics(t, func() { inf.ch <- 999 })

	// Queue not fully drained
	inf = MakeInfiniteChan[int](1)
	inf.Push(1)
	inf.Push(1)
	inf.Close(time.Millisecond * 100)
	assert.True(t, inf.closed.Load())
	assert.Panics(t, func() { inf.Push(999) })
	assert.Panics(t, func() { inf.ch <- 999 })
}

func TestUtils_InfiniteChanFastReadAfterClose(t *testing.T) {
	t.Parallel()

	n := 4
	inf := MakeInfiniteChan[int](n)
	assert.Equal(t, n, cap(inf.C()))

	// Push twice the capacity of the channel
	t0 := time.Now()
	for i := 0; i < n*2; i++ {
		inf.Push(i)
	}
	t1 := time.Now()
	assert.WithinDuration(t, t0, t1, 50*time.Millisecond, "Channel should not block")
	assert.Len(t, inf.queue, 4)
	assert.Equal(t, int32(n*2), inf.pushed.Load())

	// Close the channel
	go inf.Close(time.Second)

	// Channel should produce all elements if read quickly after closing
	m := 0
	for range inf.C() {
		m++
		time.Sleep(25 * time.Millisecond)
	}
	assert.Equal(t, n*2, m)
}

func TestUtils_InfiniteChanReadWhileOpen(t *testing.T) {
	t.Parallel()

	n := 4
	inf := MakeInfiniteChan[int](n)
	assert.Equal(t, n, cap(inf.C()))

	// Pull while channel is empty
	var wg sync.WaitGroup
	x := 0
	wg.Add(1)
	go func() {
		assert.Equal(t, 0, len(inf.ch))
		x = <-inf.C()
		assert.Equal(t, 1, x)
		wg.Done()
	}()

	time.Sleep(50 * time.Millisecond)
	inf.Push(1)
	wg.Wait()
	assert.Equal(t, 1, x)
	assert.Equal(t, 0, len(inf.ch))

	// Pull all from an overflowing channel
	for i := 0; i < n*2; i++ {
		inf.Push(i)
	}
	assert.Equal(t, n, len(inf.ch))
	assert.Equal(t, n, len(inf.queue))

	for i := 0; i < n*2; i++ {
		y := <-inf.C()
		assert.Equal(t, i, y)
	}
	assert.Equal(t, 0, len(inf.ch))
	assert.Equal(t, 0, len(inf.queue))
}

func TestUtils_InfiniteChanNoLocksBeforeCapacityReached(t *testing.T) {
	t.Parallel()

	n := 4
	inf := MakeInfiniteChan[int](n)
	assert.Equal(t, n, cap(inf.C()))
	for i := 0; i < n; i++ {
		inf.Push(i)
		assert.Zero(t, inf.locks)
		assert.Equal(t, int32(i+1), inf.pushed.Load())
	}
	inf.Push(n)
	assert.Equal(t, 1, inf.locks)
	assert.Equal(t, int32(n+1), inf.pushed.Load())
	for i := 0; i <= n; i++ {
		assert.Equal(t, i, <-inf.C())
	}
}

func BenchmarkUtils_InfiniteChanPushPull(b *testing.B) {
	inf := MakeInfiniteChan[int](128)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		inf.Push(i)
		<-inf.C()
	}

	// On 2021 MacBook Pro M1 16": 57 ns/op
}

func BenchmarkUtils_InfiniteChanPushPullUnderCapacity(b *testing.B) {
	inf := MakeInfiniteChan[int](b.N)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		inf.Push(i)
		<-inf.C()
	}

	// On 2021 MacBook Pro M1 16": 28 ns/op
}

func BenchmarkUtils_InfiniteChanPushPullStandardChan(b *testing.B) {
	ch := make(chan int, 128)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ch <- i
		<-ch
	}

	// On 2021 MacBook Pro M1 16": 27 ns/op
}

func BenchmarkUtils_InfiniteChanPushAllPullAll(b *testing.B) {
	inf := MakeInfiniteChan[int](1)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		inf.Push(i)
	}
	for i := 0; i < b.N; i++ {
		<-inf.C()
	}

	// On 2021 MacBook Pro M1 16": 54 ns/op
}
