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
	"sync"
	"unsafe"
)

var (
	pools []*sync.Pool
)

// init sets up the sized pools at 1KBK to 4MB.
func init() {
	pools = make([]*sync.Pool, 16)
	for i := range 12 + 1 {
		sizeBytes := 1 << (i + 10)
		pools[i] = &sync.Pool{
			New: func() any {
				buf := make([]byte, sizeBytes)
				return &buf
			},
		}
	}
}

/*
Alloc returns a memory block of at least the indicated byte size.
The length of the buffer is 0.

Example:

	block := mem.Alloc()
	defer mem.Free(block)
*/
func Alloc(byteSize int) []byte {
	for i := range 12 + 1 {
		if byteSize <= 1<<(i+10) {
			ptrBuf := pools[i].Get().(*[]byte)
			return (*ptrBuf)[:0]
		}
	}
	return make([]byte, byteSize)[:0]
}

// Free releases the memory block.
func Free(block []byte) {
	if block == nil {
		return
	}
	block = unsafe.Slice(unsafe.SliceData(block), cap(block)) // Reset the length
	for i := range 12 + 1 {
		if cap(block) == 1<<(i+10) {
			pools[i].Put(&block)
			return
		}
	}
}

// Copy returns a new memory block with the same contents of the original.
func Copy(original []byte) (clone []byte) {
	if original == nil {
		return nil
	}
	n := len(original)
	clone = Alloc(n)[:n]
	copy(clone, original)
	return clone
}
