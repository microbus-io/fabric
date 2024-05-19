package utils

import "sync"

// SyncMap is a type-safe version of sync.Map.
type SyncMap[K comparable, V any] struct {
	m sync.Map
}

// Load returns the value stored in the map for a key, or nil if no value is present. The ok result indicates whether value was found in the map.
func (sm *SyncMap[K, V]) Load(key K) (value V, ok bool) {
	v, ok := sm.m.Load(key)
	if !ok {
		return value, ok
	}
	return v.(V), ok
}

// Store sets the value for a key.
func (sm *SyncMap[K, V]) Store(key K, value V) {
	sm.m.Store(key, value)
}

// Delete deletes the value for a key.
func (sm *SyncMap[K, V]) Delete(key K) {
	sm.m.Delete(key)
}
