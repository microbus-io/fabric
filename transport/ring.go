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

package transport

import (
	"fmt"
	"strings"
)

// stone is an element inserted into the ring.
type stone struct {
	Handler MsgHandler
	Next    *stone
	Prev    *stone
	Index   int // Used by tests
}

// clear nullifies the next and prev links.
func (stone *stone) clear() {
	stone.Next = nil
	stone.Prev = nil
}

// ringList is a bi-directionally circular linked list.
// The ring data structure is not safe for concurrent use.
type ringList struct {
	head  *stone
	index int
	count int
}

// Insert adds a message handler to the ring and returns the stone that holds the handler.
func (r *ringList) Insert(handler MsgHandler) (gem *stone) {
	r.index++
	r.count++
	gem = &stone{
		Handler: handler,
		Index:   r.index,
	}
	if r.head == nil {
		gem.Next = gem
		gem.Prev = gem
		r.head = gem
	} else {
		// Insert before the head node
		gem.Prev = r.head.Prev
		gem.Next = r.head
		r.head.Prev.Next = gem
		r.head.Prev = gem
	}
	return gem
}

// Remove deletes the stone from the ring.
func (r *ringList) Remove(gem *stone) bool {
	if gem.Next == nil {
		return false
	}
	r.count--
	if gem.Next == gem {
		r.head = nil
		gem.clear()
		return true
	}
	gem.Prev.Next = gem.Next
	gem.Next.Prev = gem.Prev
	if r.head == gem {
		r.head = gem.Next
	}
	gem.clear()
	return true
}

// Head returns the head stone in the ring, or nil if the ring is empty.
func (r *ringList) Head() (gem *stone) {
	head := r.head
	return head
}

// Rotate changes the head of the ring to the stone following it.
// It returns head stone before the rotation, or nil if the ring is empty.
func (r *ringList) Rotate() (gem *stone) {
	if r.head == nil {
		return nil
	}
	head := r.head
	r.head = r.head.Next
	return head
}

// Len returns the number of stones in the ring.
func (r *ringList) Len() int {
	return r.count
}

// IsEmpty indicates if the ring holds no stones.
func (r *ringList) IsEmpty() bool {
	empty := r.head == nil
	return empty
}

// String prints the ring to a string. It is used by tests.
func (r *ringList) String() string {
	var sb strings.Builder
	sb.WriteString("{")
	gem := r.head
	for gem != nil {
		sb.WriteString(fmt.Sprintf("%v", gem.Index))
		gem = gem.Next
		if gem == r.head {
			break
		}
		sb.WriteString(",")
	}
	sb.WriteString("}")
	return sb.String()
}
