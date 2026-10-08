// Package stream defines the stream catalog: what a stream is, how its
// source URL is validated, and where streams are registered.
package stream

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

const (
	maxURLLength  = 2048
	maxNameLength = 64
)

var (
	ErrInvalidURL   = errors.New("invalid stream url")
	ErrInvalidName  = errors.New("invalid stream name")
	ErrNotFound     = errors.New("stream not found")
	ErrDuplicate    = errors.New("stream already exists")
	ErrLimitReached = errors.New("stream limit reached")
)

type Stream struct {
	ID        string
	Name      string
	URL       string // normalized; includes credentials when supplied
	CreatedAt time.Time
}

// Registry stores stream definitions. The context is unused by the in-memory
// implementation and exists for networked backends.
type Registry interface {
	Create(ctx context.Context, s Stream) error
	Get(ctx context.Context, id string) (Stream, error)
	List(ctx context.Context) ([]Stream, error)
	Delete(ctx context.Context, id string) error
}

// New validates the input and returns a stream with a fresh ID. An empty name
// is replaced with one derived from the URL.
func New(rawURL, name string) (Stream, error) {
	u, err := ParseURL(rawURL)
	if err != nil {
		return Stream{}, err
	}

	name = strings.TrimSpace(name)
	if name == "" {
		name = DefaultName(u)
	}
	if utf8.RuneCountInString(name) > maxNameLength {
		return Stream{}, fmt.Errorf("%w: must be at most %d characters", ErrInvalidName, maxNameLength)
	}

	return Stream{ID: NewID(), Name: name, URL: u, CreatedAt: time.Now().UTC()}, nil
}

func NewID() string {
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// ParseURL validates an rtsp:// or rtsps:// URL and returns it normalized.
func ParseURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	switch {
	case raw == "":
		return "", fmt.Errorf("%w: url is required", ErrInvalidURL)
	case len(raw) > maxURLLength:
		return "", fmt.Errorf("%w: url is longer than %d characters", ErrInvalidURL, maxURLLength)
	case strings.ContainsFunc(raw, func(r rune) bool { return unicode.IsSpace(r) || unicode.IsControl(r) }):
		return "", fmt.Errorf("%w: url must not contain spaces", ErrInvalidURL)
	}

	u, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("%w: url is malformed", ErrInvalidURL)
	}
	u.Scheme = strings.ToLower(u.Scheme)
	if u.Scheme != "rtsp" && u.Scheme != "rtsps" {
		return "", fmt.Errorf("%w: url must start with rtsp:// or rtsps://", ErrInvalidURL)
	}
	if u.Hostname() == "" {
		return "", fmt.Errorf("%w: host is missing", ErrInvalidURL)
	}
	if p := u.Port(); p != "" {
		if n, err := strconv.Atoi(p); err != nil || n < 1 || n > 65535 {
			return "", fmt.Errorf("%w: port must be between 1 and 65535", ErrInvalidURL)
		}
	}
	return u.String(), nil
}

// RedactURL masks the password so a URL can be shown to clients and logs.
func RedactURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	return u.Redacted()
}

// DefaultName is the last path segment of the URL, or its host.
func DefaultName(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	if base := path.Base(u.Path); base != "/" && base != "." {
		return base
	}
	return u.Hostname()
}
