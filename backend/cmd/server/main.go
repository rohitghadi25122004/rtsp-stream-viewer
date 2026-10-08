package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/rohit/rtsp-stream-viewer/backend/internal/config"
	"github.com/rohit/rtsp-stream-viewer/backend/internal/ffmpeg"
	"github.com/rohit/rtsp-stream-viewer/backend/internal/httpapi"
	"github.com/rohit/rtsp-stream-viewer/backend/internal/hub"
	"github.com/rohit/rtsp-stream-viewer/backend/internal/stream"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: cfg.LogLevel}))

	for _, bin := range []string{cfg.FFmpegPath, cfg.FFprobePath} {
		if _, err := exec.LookPath(bin); err != nil {
			return fmt.Errorf("%s not found; install FFmpeg or set FFMPEG_PATH and FFPROBE_PATH", bin)
		}
	}

	registry := stream.NewMemoryRegistry(cfg.MaxStreams)
	seed(registry, cfg.Seeds, log)

	src := &ffmpeg.Source{FFmpegPath: cfg.FFmpegPath, FFprobePath: cfg.FFprobePath, Log: log}
	mgr := hub.NewManager(src, hub.Config{IdleTimeout: cfg.IdleTimeout}, log)

	srv := &http.Server{
		Addr: ":" + cfg.Port,
		Handler: httpapi.NewHandler(httpapi.Options{
			Registry:       registry,
			Hub:            mgr,
			AllowedOrigins: cfg.AllowedOrigins,
			Log:            log,
		}),
		// No WriteTimeout: it would cut off long-lived WebSocket streams.
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errc := make(chan error, 1)
	go func() {
		log.Info("listening", "addr", srv.Addr, "allowed_origins", cfg.AllowedOrigins)
		errc <- srv.ListenAndServe()
	}()

	select {
	case err := <-errc:
		return err
	case <-ctx.Done():
	}

	log.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Shutdown does not wait for hijacked WebSocket connections; closing the
	// hub disconnects them and stops every FFmpeg process.
	if err := srv.Shutdown(shutdownCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Warn("http shutdown", "err", err)
	}
	return mgr.Shutdown(shutdownCtx)
}

func seed(registry stream.Registry, seeds []config.Seed, log *slog.Logger) {
	for _, sd := range seeds {
		s, err := stream.New(sd.URL, sd.Name)
		if err == nil {
			err = registry.Create(context.Background(), s)
		}
		if err != nil {
			log.Warn("skipping seed stream", "name", sd.Name, "err", err)
		}
	}
}
