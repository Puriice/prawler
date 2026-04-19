package multikey

import "sync"

// Map allows multiple keys to reference a single value.
// - O(1) average Get
// - O(1) insert
// - O(1) delete (swap with last, no holes)
// - concurrency-safe
type Map[K comparable, V comparable] struct {
	mu sync.RWMutex

	keyToIndex   map[K]int
	values       []V
	indexToKeys  map[int]map[K]struct{}
	valueToIndex map[V]int
}

// New creates a new MultiKeyMap
func New[K comparable, V comparable]() *Map[K, V] {
	return &Map[K, V]{
		keyToIndex:   make(map[K]int),
		values:       make([]V, 0),
		indexToKeys:  make(map[int]map[K]struct{}),
		valueToIndex: make(map[V]int),
	}
}

// Put inserts a value with multiple keys
// If a key already exists, it will be overwritten to point to the new value
func (m *Map[K, V]) Put(key K, value V) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 🔍 Check if value already exists
	idx, exists := m.valueToIndex[value]

	if !exists {
		// Create new entry
		idx = len(m.values)
		m.values = append(m.values, value)
		m.indexToKeys[idx] = make(map[K]struct{})
		m.valueToIndex[value] = idx
	}

	// Assign keys
	// If key already exists, remove from old group
	if oldIdx, ok := m.keyToIndex[key]; ok {
		delete(m.indexToKeys[oldIdx], key)
	}

	m.keyToIndex[key] = idx
	m.indexToKeys[idx][key] = struct{}{}
}

func (m *Map[K, V]) AddValue(value V) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// If value already exists → do nothing
	if _, exists := m.valueToIndex[value]; exists {
		return
	}

	idx := len(m.values)
	m.values = append(m.values, value)

	m.indexToKeys[idx] = make(map[K]struct{})
	m.valueToIndex[value] = idx
}

// Get retrieves a value by key
func (m *Map[K, V]) Get(key K) (V, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	idx, ok := m.keyToIndex[key]
	if !ok {
		var zero V
		return zero, false
	}

	return m.values[idx], true
}

func (m *Map[K, V]) GetValueWithLeastKey() (V, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.values) == 0 {
		var zero V
		return zero, false
	}

	minIdx := -1
	minLoad := int(^uint(0) >> 1)

	for idx, keys := range m.indexToKeys {
		if len(keys) < minLoad {
			minLoad = len(keys)
			minIdx = idx
		}
	}

	if minIdx == -1 {
		var zero V
		return zero, false
	}

	return m.values[minIdx], true
}

func (m *Map[K, V]) removeByIndex(idx int) {
	lastIdx := len(m.values) - 1
	val := m.values[idx]

	// 1. Remove all keys
	for k := range m.indexToKeys[idx] {
		delete(m.keyToIndex, k)
	}

	// 2. Remove value mapping
	delete(m.valueToIndex, val)

	// 3. Swap if needed
	if idx != lastIdx {
		lastVal := m.values[lastIdx]

		m.values[idx] = lastVal

		for k := range m.indexToKeys[lastIdx] {
			m.keyToIndex[k] = idx
		}

		m.indexToKeys[idx] = m.indexToKeys[lastIdx]

		// 🔁 FIX: update moved value index
		m.valueToIndex[lastVal] = idx
	}

	// 4. Cleanup
	m.values = m.values[:lastIdx]
	delete(m.indexToKeys, lastIdx)
}

func (m *Map[K, V]) RemoveKey(key K) {
	m.mu.Lock()
	defer m.mu.Unlock()

	idx, ok := m.keyToIndex[key]
	if !ok {
		return
	}

	delete(m.keyToIndex, key)
	delete(m.indexToKeys[idx], key)
}

func (m *Map[K, V]) RemoveByKey(key K) {
	m.mu.Lock()
	defer m.mu.Unlock()

	idx, ok := m.keyToIndex[key]
	if !ok {
		return
	}

	delete(m.keyToIndex, key)
	delete(m.indexToKeys[idx], key)

	if len(m.indexToKeys[idx]) > 0 {
		return
	}

	m.removeByIndex(idx)
}

func (m *Map[K, V]) RemoveByValue(value V) {
	m.mu.Lock()
	defer m.mu.Unlock()

	idx, ok := m.valueToIndex[value]
	if !ok {
		return
	}

	m.removeByIndex(idx)
}

// Len returns number of stored values (not keys)
func (m *Map[K, V]) Len() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.values)
}

// Has checks if a key exists
func (m *Map[K, V]) HasKey(key K) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	_, ok := m.keyToIndex[key]
	return ok
}

func (m *Map[K, V]) HasValue(value V) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	_, ok := m.valueToIndex[value]
	return ok
}
