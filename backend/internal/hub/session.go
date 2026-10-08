// Package hub runs one source pipeline per stream and shares its output
// with every viewer. A pipeline starts with the first viewer, stops after
// the last one has been gone for a while, and restarts with backoff on
// failure.
package hub

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/rohit/rtsp-stream-viewer/backend/internal/fmp4"
)

// Source produces fMP4 segments for a stream URL until it fails or ctx is
// cancelled. Callbacks are invoked from the goroutine that called Run.
type Source interface {
	Run(ctx context.Context, url string, onInit func(fmp4.Init), onFragment func([]byte)) error
}

type Config struct {
	IdleTimeout      time.Duration
	BackoffMin       time.Duration
	BackoffMax       time.Duration
	SubscriberBuffer int
}

func (c Config) withDefaults() Config {
	if c.IdleTimeout <= 0 {
		c.IdleTimeout = 30 * time.Second
	}
	if c.BackoffMin <= 0 {
		c.BackoffMin = time.Second
	}
	if c.BackoffMax <= 0 {
		c.BackoffMax = 30 * time.Second
	}
	if c.SubscriberBuffer <= 0 {
		c.SubscriberBuffer = 32
	}
	return c
}

// sourceError is implemented by source failures that can describe
// themselves to viewers.
type sourceError interface {
	Code() string
	Message() string
	Retryable() bool
}

// session owns the pipeline of one stream. Fields below mu are guarded by
// it, and it is never held across a blocking operation.
type session struct {
	id, url string
	src     Source
	cfg     Config
	log     *slog.Logger

	mu     sync.Mutex
	subs   map[*Subscriber]struct{}
	status Status
	init   *fmp4.Init
	// last is the newest fragment. Fragments start on a keyframe, so a new
	// viewer can begin playback from it right away.
	last      []byte
	running   bool
	cancel    context.CancelFunc
	gen       uint64 // bumped on every start and stop, so callbacks from an old pipeline are ignored
	idleTimer *time.Timer
	closed    bool

	wg sync.WaitGroup
}

func newSession(id, url string, src Source, cfg Config, log *slog.Logger) *session {
	return &session{
		id:     id,
		url:    url,
		src:    src,
		cfg:    cfg,
		log:    log.With("stream_id", id),
		subs:   make(map[*Subscriber]struct{}),
		status: Status{State: StateIdle},
	}
}

func (s *session) subscribe() (*Subscriber, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil, ErrStreamRemoved
	}
	s.stopIdleTimerLocked()
	if !s.running {
		s.startLocked()
	}

	sub := newSubscriber(s, s.cfg.SubscriberBuffer)
	sub.ch <- Message{Kind: MsgStatus, Status: s.status}
	if s.init != nil {
		sub.ch <- Message{Kind: MsgInit, Codec: s.init.Codec, Data: s.init.Data}
		if s.last != nil {
			sub.ch <- Message{Kind: MsgFragment, Data: s.last}
		}
	}
	s.subs[sub] = struct{}{}
	return sub, nil
}

func (s *session) unsubscribe(sub *Subscriber) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.subs[sub]; ok {
		s.removeLocked(sub)
	}
}

func (s *session) snapshot() (Status, int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.status, len(s.subs)
}

// close stops the pipeline, disconnects every subscriber with reason and
// waits for the pipeline goroutine to exit.
func (s *session) close(reason error) {
	s.mu.Lock()
	s.closed = true
	s.stopIdleTimerLocked()
	s.stopLocked()
	s.setStatusLocked(Status{State: StateStopped})
	for sub := range s.subs {
		delete(s.subs, sub)
		sub.finish(reason)
	}
	s.mu.Unlock()

	s.wg.Wait()
}

func (s *session) stopIfIdle() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.idleTimer = nil
	if len(s.subs) == 0 && s.running && !s.closed {
		s.log.Info("stopping idle pipeline")
		s.stopLocked()
	}
}

func (s *session) startLocked() {
	s.gen++
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	s.running = true
	s.init, s.last = nil, nil
	s.setStatusLocked(Status{State: StateConnecting})

	s.wg.Add(1)
	go s.run(ctx, s.gen)
}

func (s *session) stopLocked() {
	if !s.running {
		return
	}
	s.cancel()
	s.gen++
	s.running = false
	s.init, s.last = nil, nil
	s.status = Status{State: StateIdle}
}

func (s *session) stopIdleTimerLocked() {
	if s.idleTimer != nil {
		s.idleTimer.Stop()
		s.idleTimer = nil
	}
}

func (s *session) run(ctx context.Context, gen uint64) {
	defer s.wg.Done()

	b := &backoff{min: s.cfg.BackoffMin, max: s.cfg.BackoffMax}
	for attempt := 1; ; attempt++ {
		err := s.src.Run(ctx, s.url, s.initHandler(gen), s.fragmentHandler(gen, b))
		if ctx.Err() != nil {
			return
		}

		code, message, retry := "exited", "Stream ended unexpectedly", true
		var se sourceError
		if errors.As(err, &se) {
			code, message, retry = se.Code(), se.Message(), se.Retryable()
		}

		if !retry {
			s.log.Warn("pipeline failed", "err", err)
			s.finish(gen, Status{State: StateError, Code: code, Message: message})
			return
		}

		delay := b.Next()
		s.log.Warn("pipeline failed, retrying", "err", err, "attempt", attempt, "retry_in", delay)
		s.setStatus(gen, Status{State: StateReconnecting, Code: code, Message: message, RetryInMs: delay.Milliseconds()})

		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return
		}
		s.setStatus(gen, Status{State: StateConnecting})
	}
}

// finish retires a pipeline that gave up on its own. Pipelines stopped from
// outside have already been retired by stopLocked.
func (s *session) finish(gen uint64, final Status) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if gen != s.gen {
		return
	}
	s.cancel()
	s.running = false
	s.init, s.last = nil, nil
	s.setStatusLocked(final)
}

// setStatus publishes a state change from the pipeline. Any state other
// than live drops the cached media, since the next init replaces it.
func (s *session) setStatus(gen uint64, st Status) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if gen != s.gen {
		return
	}
	s.init, s.last = nil, nil
	s.setStatusLocked(st)
}

func (s *session) setStatusLocked(st Status) {
	if st == s.status {
		return
	}
	if st.State != s.status.State {
		s.log.Info("state changed", "state", st.State, "viewers", len(s.subs))
	}
	s.status = st
	s.broadcastLocked(Message{Kind: MsgStatus, Status: st})
}

func (s *session) initHandler(gen uint64) func(fmp4.Init) {
	return func(init fmp4.Init) {
		s.mu.Lock()
		defer s.mu.Unlock()

		if gen != s.gen {
			return
		}
		s.init, s.last = &init, nil
		s.broadcastLocked(Message{Kind: MsgInit, Codec: init.Codec, Data: init.Data})
	}
}

func (s *session) fragmentHandler(gen uint64, b *backoff) func([]byte) {
	return func(frag []byte) {
		s.mu.Lock()
		defer s.mu.Unlock()

		if gen != s.gen {
			return
		}
		s.last = frag
		if s.status.State != StateLive {
			b.Reset()
			s.setStatusLocked(Status{State: StateLive})
		}
		s.broadcastLocked(Message{Kind: MsgFragment, Data: frag})
	}
}

// broadcastLocked delivers msg to every subscriber without blocking. A
// subscriber whose buffer is full is dropped rather than slowing the others.
func (s *session) broadcastLocked(msg Message) {
	for sub := range s.subs {
		select {
		case sub.ch <- msg:
		default:
			s.log.Warn("dropping slow subscriber")
			s.removeLocked(sub)
			sub.finish(ErrSlowConsumer)
		}
	}
}

func (s *session) removeLocked(sub *Subscriber) {
	delete(s.subs, sub)
	if len(s.subs) == 0 && s.running && !s.closed && s.idleTimer == nil {
		s.idleTimer = time.AfterFunc(s.cfg.IdleTimeout, s.stopIfIdle)
	}
}
