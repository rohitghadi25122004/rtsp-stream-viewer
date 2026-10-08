package hub

import (
	"context"
	"log/slog"
	"sync"
)

// Manager maps stream IDs to sessions. Lock order is Manager.mu before
// session.mu; sessions never call back into the manager.
type Manager struct {
	src Source
	cfg Config
	log *slog.Logger

	mu       sync.Mutex
	sessions map[string]*session
	closed   bool
}

func NewManager(src Source, cfg Config, log *slog.Logger) *Manager {
	return &Manager{
		src:      src,
		cfg:      cfg.withDefaults(),
		log:      log,
		sessions: make(map[string]*session),
	}
}

// Subscribe attaches a viewer to the stream, starting its pipeline if needed.
func (m *Manager) Subscribe(id, url string) (*Subscriber, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return nil, ErrShuttingDown
	}
	s, ok := m.sessions[id]
	if !ok {
		s = newSession(id, url, m.src, m.cfg, m.log)
		m.sessions[id] = s
	}
	return s.subscribe()
}

// Status reports the pipeline state and viewer count of a stream.
func (m *Manager) Status(id string) (Status, int) {
	m.mu.Lock()
	s, ok := m.sessions[id]
	m.mu.Unlock()

	if !ok {
		return Status{State: StateIdle}, 0
	}
	return s.snapshot()
}

// Remove stops the stream's pipeline and disconnects its viewers.
func (m *Manager) Remove(id string) {
	m.mu.Lock()
	s, ok := m.sessions[id]
	delete(m.sessions, id)
	m.mu.Unlock()

	if ok {
		s.close(ErrStreamRemoved)
	}
}

// Shutdown disconnects all viewers and waits for every pipeline to exit.
func (m *Manager) Shutdown(ctx context.Context) error {
	m.mu.Lock()
	m.closed = true
	sessions := m.sessions
	m.sessions = nil
	m.mu.Unlock()

	done := make(chan struct{})
	go func() {
		var wg sync.WaitGroup
		for _, s := range sessions {
			wg.Go(func() { s.close(ErrShuttingDown) })
		}
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
