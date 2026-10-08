package ffmpeg

import (
	"bytes"
	"cmp"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os/exec"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rohit/rtsp-stream-viewer/backend/internal/fmp4"
)

const (
	probeTimeout = 10 * time.Second

	// The first fragment can take a while: RTSP handshake, then a full GOP.
	// After that a healthy camera produces a fragment at every keyframe.
	firstFragmentTimeout = 20 * time.Second
	fragmentTimeout      = 15 * time.Second
)

// Source runs one FFmpeg process per call to Run and turns its output into
// fMP4 segments.
type Source struct {
	FFmpegPath  string
	FFprobePath string
	Log         *slog.Logger
}

// Run probes the source, starts FFmpeg, and delivers segments until the
// stream fails or ctx is cancelled. onInit is called once, before any
// fragment. Callbacks run on the calling goroutine and must not block.
func (s *Source) Run(ctx context.Context, url string, onInit func(fmp4.Init), onFragment func([]byte)) error {
	log := s.Log.With("url", redact(url))

	probeCtx, cancelProbe := context.WithTimeout(ctx, probeTimeout)
	probe, err := Probe(probeCtx, s.FFprobePath, url)
	cancelProbe()
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if err != nil {
		return err
	}

	mode := probe.Mode()
	log.Info("starting ffmpeg", "codec", probe.Codec, "profile", probe.Profile, "mode", mode)

	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	stderr := &lineWriter{onLine: func(line string) { log.Debug("ffmpeg", "line", redact(line)) }}
	cmd := exec.CommandContext(runCtx, s.FFmpegPath, StreamArgs(url, mode)...)
	cmd.Stderr = stderr
	cmd.WaitDelay = 3 * time.Second
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start ffmpeg: %w", err)
	}

	var stalled atomic.Bool
	watchdog := time.AfterFunc(firstFragmentTimeout, func() {
		stalled.Store(true)
		cancel()
	})
	defer watchdog.Stop()

	readErr := segment(stdout, watchdog, onInit, onFragment)
	cancel()
	waitErr := cmd.Wait()

	switch {
	case ctx.Err() != nil:
		return ctx.Err()
	case stalled.Load():
		return &Error{Kind: KindStalled, Detail: "No video received from the camera", Err: readErr}
	case readErr != nil && !errors.Is(readErr, io.EOF) && !errors.Is(readErr, io.ErrUnexpectedEOF):
		return &Error{Kind: KindInvalidData, Detail: detail(readErr.Error()), Err: readErr}
	}
	return errorFromOutput(stderr.tail(), cmp.Or(waitErr, readErr))
}

func segment(r io.Reader, watchdog *time.Timer, onInit func(fmp4.Init), onFragment func([]byte)) error {
	seg := fmp4.NewSegmenter(r)
	initSeg, err := seg.ReadInit()
	if err != nil {
		return err
	}
	onInit(initSeg)

	for {
		frag, err := seg.ReadFragment()
		if err != nil {
			return err
		}
		watchdog.Reset(fragmentTimeout)
		onFragment(frag)
	}
}

// lineWriter splits FFmpeg's stderr into lines and keeps the most recent
// ones for diagnosis after the process exits.
type lineWriter struct {
	onLine func(string)

	mu      sync.Mutex
	partial []byte
	lines   [][]byte
}

const keepLines = 20

func (w *lineWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.partial = append(w.partial, p...)
	for {
		i := bytes.IndexAny(w.partial, "\r\n")
		if i < 0 {
			break
		}
		if line := bytes.TrimSpace(w.partial[:i]); len(line) > 0 {
			w.onLine(string(line))
			w.lines = append(w.lines, bytes.Clone(line))
			if len(w.lines) > keepLines {
				w.lines = w.lines[1:]
			}
		}
		w.partial = w.partial[i+1:]
	}
	return len(p), nil
}

func (w *lineWriter) tail() []byte {
	w.mu.Lock()
	defer w.mu.Unlock()

	return bytes.Join(append(slices.Clone(w.lines), w.partial), []byte("\n"))
}
