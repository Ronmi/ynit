package main

import "sync"

// SyncIntMap is synchronized map[int]bool
type SyncIntMap struct {
	data map[int]bool
	lock *sync.Mutex
}

// NewIntMap creates a new SyncIntMap
func NewIntMap() *SyncIntMap {
	return &SyncIntMap{
		map[int]bool{},
		&sync.Mutex{},
	}
}

// Has a key in this map?
func (m *SyncIntMap) Has(key int) bool {
	m.lock.Lock()
	defer m.lock.Unlock()
	return m.data[key]
}

// Set the value of the key in this map
func (m *SyncIntMap) Set(key int, val bool) {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.data[key] = val
}
