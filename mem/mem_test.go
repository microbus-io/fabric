/*
Copyright (c) 2023-2025 Microbus LLC and various contributors

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

package mem

import (
	"bytes"
	"testing"
	"unsafe"

	"github.com/microbus-io/testarossa"
)

func TestMem_Recycle(t *testing.T) {
	// No parallel
	assert := testarossa.For(t)

	block1 := Alloc(1024)
	assert.Equal(0, len(block1))
	assert.Equal(1024, cap(block1))
	block1[:1][0] = '1'
	assert.Equal([]byte("1"), block1[:1])

	block2 := Alloc(1024)
	assert.Equal(0, len(block2))
	assert.Equal(1024, cap(block2))
	block2[:1][0] = '2'
	assert.Equal([]byte("2"), block2[:1])
	assert.NotEqual([]byte("2"), block1[:1])

	Free(block1)

	block3 := Alloc(1024)
	assert.Equal(0, len(block3))
	assert.Equal(1024, cap(block3))
	assert.Equal(unsafe.SliceData(block1), unsafe.SliceData(block3))
	assert.Equal([]byte("1"), block1[:1])
}

func TestMem_Grow(t *testing.T) {
	// No parallel
	assert := testarossa.For(t)

	block1 := Alloc(2<<10 + 1)
	assert.Len(block1, 0)
	assert.Equal(4<<10, cap(block1))

	buf := append(block1, bytes.Repeat([]byte{'1'}, 1<<10)...)
	assert.Len(buf, 1<<10)
	assert.Equal(4<<10, cap(buf))
	assert.Equal(byte('1'), block1[:1][0])
	buf[0] = '2'
	assert.Equal(byte('2'), block1[:1][0])

	buf = append(buf, bytes.Repeat([]byte{'1'}, 4<<10)...)
	assert.Len(buf, 5<<10)
	assert.True(cap(buf) >= 5<<10)
	assert.Len(block1, 0)
	assert.Equal(byte('2'), block1[:1][0])
	buf[0] = '3'
	assert.Equal(byte('2'), block1[:1][0])

	buf = nil
	Free(block1)

	block2 := Alloc(2<<10 + 2)
	assert.Len(block2, 0)
	assert.Equal(4<<10, cap(block2))
	assert.Equal(byte('2'), block2[:1][0])
}

func TestMem_TooLarge(t *testing.T) {
	// No parallel
	assert := testarossa.For(t)

	block1 := Alloc(1 << (12 + 1 + 10 + 1)) // 8MB
	block1 = append(block1, []byte("X2865374563X")...)
	Free(block1)
	block2 := Alloc(1 << (12 + 1 + 10 + 1)) // Same 8MB
	assert.NotEqual([]byte("X2865374563X"), block2[:12])
	Free(block2)
}

func TestMem_TooLargeZeroLength(t *testing.T) {
	// No parallel
	assert := testarossa.For(t)

	// A request larger than the largest pool class falls through to a plain allocation, but must still come back
	// zero-length with capacity for the request - exactly like the pooled path - so append starts at index 0
	// rather than after a run of zero bytes.
	const size = 8 << 20 // 8MB, above the 4MB largest class
	block := Alloc(size)
	assert.Equal(0, len(block))
	assert.Equal(size, cap(block))

	block = append(block, []byte("hello")...)
	assert.Equal([]byte("hello"), block)
}

func TestMem_Copy(t *testing.T) {
	// No parallel
	assert := testarossa.For(t)

	src := Alloc(1<<10 + 16)
	src = append(src, bytes.Repeat([]byte{'1'}, 1<<10)...)

	dest := Copy(src)
	assert.Equal(src, dest)
	assert.Equal(len(src), len(dest))

	src[0] = byte('2')
	assert.Equal(byte('1'), dest[0])

	Free(src)
	assert.Equal(bytes.Repeat([]byte{'1'}, 1<<10), dest)
	Free(dest)
}
