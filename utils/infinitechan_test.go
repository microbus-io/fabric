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
	assert.Equal(t, n*2, inf.count)

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
	assert.Equal(t, n*2, inf.count)

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

func BenchmarkUtils_InfiniteChanPushPull(b *testing.B) {
	inf := MakeInfiniteChan[int](128)
	for i := 0; i < b.N; i++ {
		inf.Push(i)
		<-inf.C()
	}

	// On 2021 MacBook Pro M1 16": 52 ns/op
}
