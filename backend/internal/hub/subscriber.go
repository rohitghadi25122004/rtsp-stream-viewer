package hub

import (
	"errors"
	"sync"
)

var (
	ErrSlowConsumer  = errors.New("subscriber too slow")
	ErrStreamRemoved = errors.New("stream removed")
	ErrShuttingDown  = errors.New("server shutting down")
)

type MessageKind int

const (
	MsgStatus MessageKind = iota
	MsgInit
	MsgFragment
)

// Message is one item in a subscriber's feed. Data is shared between
// subscribers and must not be modified.
type Message struct {
	Kind   MessageKind
	Status Status
	Codec  string
	Data   []byte
}

// Subscriber receives a single viewer's copy of a session's output.
type Subscriber struct {
	ch      chan Message
	done    chan struct{}
	once    sync.Once
	err     error
	session *session
}

func newSubscriber(s *session, buffer int) *Subscriber {
	return &Subscriber{
		ch:      make(chan Message, buffer),
		done:    make(chan struct{}),
		session: s,
	}
}

func (s *Subscriber) Messages() <-chan Message { return s.ch }

// Done is closed when the subscriber is evicted or the stream goes away.
func (s *Subscriber) Done() <-chan struct{} { return s.done }

// Err reports why Done was closed. It is only valid after Done is closed.
func (s *Subscriber) Err() error { return s.err }

// Close unsubscribes. It is safe to call more than once.
func (s *Subscriber) Close() {
	s.session.unsubscribe(s)
	s.finish(nil)
}

// finish records the reason and closes done. The message channel is never
// closed: sends happen under the session lock and closing would race them.
func (s *Subscriber) finish(err error) {
	s.once.Do(func() {
		s.err = err
		close(s.done)
	})
}
