package set

import "sync"

type Set[K comparable] struct {
	m  map[K]struct{}
	mu sync.RWMutex
}

func NewSet[K comparable](init ...K) *Set[K] {
	set := &Set[K]{
		m: make(map[K]struct{}, len(init)),
	}

	for _, value := range init {
		set.m[value] = struct{}{}
	}

	return set
}

func (s *Set[K]) Add(value K) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.m[value] = struct{}{}
}

func (s *Set[K]) Remove(value K) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.m, value)
}

func (s *Set[K]) Contains(value K) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	_, exist := s.m[value]

	return exist
}
