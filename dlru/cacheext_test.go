/*
Copyright (c) 2023-2026 Microbus LLC and various contributors

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

package dlru

import (
	"slices"

	"github.com/microbus-io/seamster"
)

// FaultSkipLeave is the fault that makes Close skip the leave announcement. Test-only.
const FaultSkipLeave = faultSkipLeave

// FaultStaleGen is the fault that makes the next outgoing Store or Load carry a stale generation. Test-only.
const FaultStaleGen = faultStaleGen

// FaultSkipOffload is the fault that makes offload return without shedding. Test-only.
const FaultSkipOffload = faultSkipOffload

// CheckpointOffloadBeforeStore fires in offload before a displaced key's soft store. Test-only.
const CheckpointOffloadBeforeStore = checkpointOffloadBeforeStore

// CheckpointBeforeSend fires in Store after the request is stamped but before it is sent. Test-only.
const CheckpointBeforeSend = checkpointBeforeSend

// Members returns a snapshot of the membership set. Test-only.
func (c *Cache) Members() []string {
	c.mux.RLock()
	defer c.mux.RUnlock()
	return slices.Clone(c.members)
}

// Generation returns the current generation hash. Test-only.
func (c *Cache) Generation() string {
	c.mux.RLock()
	defer c.mux.RUnlock()
	return c.generation
}

// OwnerOf returns the computed HRW owner of a key. Test-only.
func (c *Cache) OwnerOf(key string) string {
	return c.ownerOf(key)
}

// Seams exposes the cache's fault-injection seams so tests can arm any fault. Test-only.
func (c *Cache) Seams() *seamster.Seamster {
	return c.seams
}
