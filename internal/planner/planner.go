package planner

import (
	"sync"

	"github.com/purrice/prawler/internal/multikey"
)

type Planner[K comparable, V comparable] struct {
	register *multikey.Map[K, V]
	mu       sync.Mutex
}

func NewPlanner[K comparable, V comparable]() *Planner[K, V] {
	return &Planner[K, V]{
		register: multikey.New[K, V](),
	}
}

func (p *Planner[K, V]) AddResource(resource V) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.register.AddValue(resource)
}

func (p *Planner[K, V]) RemoveResource(resource V) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.register.RemoveByValue(resource)
}

func (p *Planner[K, V]) Assign(key K, resource V) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.register.Put(key, resource)
}

func (p *Planner[K, V]) Plan(key K) (V, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	resource, ok := p.register.Get(key)

	if ok {
		return resource, true
	}

	resource, ok = p.register.GetValueWithLeastKey()

	if !ok {
		var zero V
		return zero, false
	}

	p.register.Put(key, resource)

	return resource, true
}

func (p *Planner[K, V]) Done(key K) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.register.RemoveKey(key)
}
