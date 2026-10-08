package stream

import (
	"context"
	"slices"
	"sync"
)

// MemoryRegistry keeps streams in process memory, in creation order.
type MemoryRegistry struct {
	mu      sync.RWMutex
	streams []Stream
	limit   int
}

func NewMemoryRegistry(limit int) *MemoryRegistry {
	return &MemoryRegistry{limit: limit}
}

func (m *MemoryRegistry) Create(_ context.Context, s Stream) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if slices.ContainsFunc(m.streams, func(e Stream) bool { return e.URL == s.URL }) {
		return ErrDuplicate
	}
	if m.limit > 0 && len(m.streams) >= m.limit {
		return ErrLimitReached
	}
	m.streams = append(m.streams, s)
	return nil
}

func (m *MemoryRegistry) Get(_ context.Context, id string) (Stream, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if i := m.index(id); i >= 0 {
		return m.streams[i], nil
	}
	return Stream{}, ErrNotFound
}

func (m *MemoryRegistry) List(_ context.Context) ([]Stream, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return slices.Clone(m.streams), nil
}

func (m *MemoryRegistry) Delete(_ context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	i := m.index(id)
	if i < 0 {
		return ErrNotFound
	}
	m.streams = slices.Delete(m.streams, i, i+1)
	return nil
}

func (m *MemoryRegistry) index(id string) int {
	return slices.IndexFunc(m.streams, func(s Stream) bool { return s.ID == id })
}
