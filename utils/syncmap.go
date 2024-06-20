/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package utils

import (
	"sync"
)

// SyncMap is a map wrapped in a mutex.
type SyncMap[K comparable, V any] struct {
	m   map[K]V
	mux sync.Mutex
}

// Load returns the value stored in the map for a key, or nil if no value is present. The ok result indicates whether value was found in the map.
func (sm *SyncMap[K, V]) Load(key K) (value V, ok bool) {
	sm.mux.Lock()
	defer sm.mux.Unlock()
	if sm.m == nil {
		return value, false
	}
	value, ok = sm.m[key]
	return value, ok
}

// Store sets the value for a key.
func (sm *SyncMap[K, V]) Store(key K, value V) {
	sm.mux.Lock()
	defer sm.mux.Unlock()
	if sm.m == nil {
		sm.m = make(map[K]V, 128)
	}
	sm.m[key] = value
}

// Delete deletes the value for a key.
func (sm *SyncMap[K, V]) Delete(key K) {
	sm.mux.Lock()
	defer sm.mux.Unlock()
	if sm.m == nil {
		return
	}
	delete(sm.m, key)
}
