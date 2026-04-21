package main

import (
	"sync"

	"github.com/bits-and-blooms/bloom/v3"
)

type BloomFilter struct {
	bloom bloom.BloomFilter
	mu    sync.Mutex
}

func NewBloom(n uint, p float64) *BloomFilter {
	return &BloomFilter{
		bloom: *bloom.NewWithEstimates(n, p),
	}
}

func (b *BloomFilter) Add(uri string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.bloom.AddString(uri)
}

func (b *BloomFilter) Contains(uri string) bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.bloom.TestString(uri)
}
