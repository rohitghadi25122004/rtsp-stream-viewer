package config

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Port           string
	AllowedOrigins []string
	FFmpegPath     string
	FFprobePath    string
	IdleTimeout    time.Duration
	MaxStreams     int
	LogLevel       slog.Level
	Seeds          []Seed
}

// Seed is a stream registered at startup, so a fresh instance has
// something to show.
type Seed struct {
	Name string
	URL  string
}

func Load() (Config, error) {
	cfg := Config{
		Port:           env("PORT", "8080"),
		AllowedOrigins: list(env("ALLOWED_ORIGINS", "http://localhost:5173")),
		FFmpegPath:     env("FFMPEG_PATH", "ffmpeg"),
		FFprobePath:    env("FFPROBE_PATH", "ffprobe"),
	}

	var err error
	if cfg.IdleTimeout, err = time.ParseDuration(env("IDLE_TIMEOUT", "30s")); err != nil || cfg.IdleTimeout <= 0 {
		return Config{}, fmt.Errorf("IDLE_TIMEOUT: must be a positive duration such as 30s")
	}
	if cfg.MaxStreams, err = strconv.Atoi(env("MAX_STREAMS", "16")); err != nil || cfg.MaxStreams <= 0 {
		return Config{}, fmt.Errorf("MAX_STREAMS: must be a positive integer")
	}
	if err := cfg.LogLevel.UnmarshalText([]byte(env("LOG_LEVEL", "info"))); err != nil {
		return Config{}, fmt.Errorf("LOG_LEVEL: must be debug, info, warn or error")
	}
	if cfg.Seeds, err = parseSeeds(os.Getenv("SEED_STREAMS")); err != nil {
		return Config{}, fmt.Errorf("SEED_STREAMS: %w", err)
	}
	return cfg, nil
}

// parseSeeds reads "Name=rtsp://host/a,Other=rtsp://host/b". Entries are
// split on the first '=' so query strings in the URL survive.
func parseSeeds(raw string) ([]Seed, error) {
	var seeds []Seed
	for _, entry := range list(raw) {
		name, url, ok := strings.Cut(entry, "=")
		if !ok || strings.TrimSpace(url) == "" {
			return nil, fmt.Errorf("entry %q must look like Name=rtsp://host/path", entry)
		}
		seeds = append(seeds, Seed{Name: strings.TrimSpace(name), URL: strings.TrimSpace(url)})
	}
	return seeds, nil
}

func env(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func list(raw string) []string {
	var out []string
	for item := range strings.SplitSeq(raw, ",") {
		if item = strings.TrimSpace(item); item != "" {
			out = append(out, item)
		}
	}
	return out
}
