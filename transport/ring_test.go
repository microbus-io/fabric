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
	"testing"

	"github.com/microbus-io/testarossa"
)

func TestTransport_Ring(t *testing.T) {
	t.Parallel()
	assert := testarossa.For(t)

	var ring ringList
	called := make([]bool, 5)
	h1 := func(msg *Msg) { called[1] = true }
	h2 := func(msg *Msg) { called[2] = true }
	h3 := func(msg *Msg) { called[3] = true }
	h4 := func(msg *Msg) { called[4] = true }

	// Insert
	assert.True(ring.IsEmpty())
	gem1 := ring.Insert(h1)
	gem2 := ring.Insert(h2)
	gem3 := ring.Insert(h3)
	assert.Equal("{1,2,3}", ring.String())
	assert.False(ring.IsEmpty())

	assert.False(called[1])
	gem1.Handler(nil)
	assert.True(called[1])
	assert.False(called[2])
	gem2.Handler(nil)
	assert.True(called[2])
	assert.False(called[3])
	gem3.Handler(nil)
	assert.True(called[3])

	// Rotate
	assert.Equal(gem1, ring.Head())
	assert.Equal(gem1, ring.Rotate())
	assert.Equal(gem2, ring.Head())

	assert.Equal("{2,3,1}", ring.String())
	assert.Equal(gem2, ring.Rotate())
	assert.Equal("{3,1,2}", ring.String())
	assert.Equal(gem3, ring.Rotate())
	assert.Equal("{1,2,3}", ring.String())
	assert.Equal(gem1, ring.Rotate())
	assert.Equal("{2,3,1}", ring.String())

	// Remove
	assert.True(ring.Remove(gem3))
	assert.Equal("{2,1}", ring.String())
	assert.False(ring.Remove(gem3))
	assert.True(ring.Remove(gem1))
	assert.Equal("{2}", ring.String())
	assert.True(ring.Remove(gem2))
	assert.Equal("{}", ring.String())
	assert.True(ring.IsEmpty())

	// Rotate empty ring
	assert.Nil(ring.Rotate())
	assert.Equal("{}", ring.String())

	// Rotate single gem
	gem4 := ring.Insert(h4)
	assert.Equal("{4}", ring.String())
	assert.Equal(gem4, ring.Rotate())
	assert.Equal("{4}", ring.String())
}

func TestTransport_Remove(t *testing.T) {
	t.Parallel()
	assert := testarossa.For(t)

	var ring ringList
	h1 := func(msg *Msg) {}
	h2 := func(msg *Msg) {}
	h3 := func(msg *Msg) {}
	h4 := func(msg *Msg) {}

	// Populate the ring
	gem1 := ring.Insert(h1)
	gem2 := ring.Insert(h2)
	gem3 := ring.Insert(h3)
	gem4 := ring.Insert(h4)

	assert.Equal(gem1, ring.Head())
	assert.Equal("{1,2,3,4}", ring.String())
	assert.Equal(4, ring.Len())
	assert.Equal(gem1, ring.Head())

	// Delete non-head gem
	ring.Remove(gem3)
	assert.Equal(gem1, ring.Head())
	assert.Equal("{1,2,4}", ring.String())
	assert.Equal(gem1, ring.Head())

	// Delete head node
	ring.Remove(gem1)
	assert.Equal(gem2, ring.Head())
	assert.Equal("{2,4}", ring.String())
	assert.Equal(gem2, ring.Head())

	assert.Equal(2, ring.Len())

	// Deleting a deleted gem is a no-op
	assert.False(ring.Remove(gem1))
	assert.Equal(2, ring.Len())

	// Delete all nodes
	ring.Remove(gem2)
	ring.Remove(gem4)
	assert.Zero(ring.Len())
	assert.Equal("{}", ring.String())
	assert.Nil(ring.Head())
	ring.Rotate()
	assert.Nil(ring.Head())
}
