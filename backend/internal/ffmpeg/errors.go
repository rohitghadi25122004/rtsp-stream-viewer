package ffmpeg

import (
	"bytes"
	"regexp"
	"strings"
)

// Kind classifies why a source failed. Values are sent to clients as codes.
type Kind string

const (
	KindUnreachable  Kind = "unreachable"
	KindUnauthorized Kind = "unauthorized"
	KindNotFound     Kind = "not_found"
	KindNoVideo      Kind = "no_video"
	KindInvalidData  Kind = "invalid_data"
	KindStalled      Kind = "stalled"
	KindExited       Kind = "exited"
)

// Error is a source failure with a redacted, client-safe detail line.
type Error struct {
	Kind   Kind
	Detail string
	Err    error
}

func (e *Error) Error() string {
	if e.Detail != "" {
		return string(e.Kind) + ": " + e.Detail
	}
	return string(e.Kind)
}

func (e *Error) Unwrap() error { return e.Err }

func (e *Error) Code() string { return string(e.Kind) }

func (e *Error) Message() string { return e.Detail }

// Retryable is false when retrying cannot help without a change on the
// camera side.
func (e *Error) Retryable() bool {
	return e.Kind != KindUnauthorized && e.Kind != KindNoVideo
}

var classifiers = []struct {
	kind     Kind
	patterns []string
}{
	{KindUnauthorized, []string{"401 unauthorized", "authorization failed"}},
	{KindNotFound, []string{"404 not found"}},
	{KindInvalidData, []string{"invalid data found when processing input"}},
	{KindUnreachable, []string{
		"connection refused",
		"connection reset",
		"error number -10054",
		"connection timed out",
		"timed out",
		"no route to host",
		"name or service not known",
		"temporary failure in name resolution",
		"network is unreachable",
		"i/o timeout",
		"i/o error",
		"error number -138",
	}},
}

func classify(line string) (Kind, bool) {
	lower := strings.ToLower(line)
	for _, c := range classifiers {
		for _, p := range c.patterns {
			if strings.Contains(lower, p) {
				return c.kind, true
			}
		}
	}
	return "", false
}

var credentials = regexp.MustCompile(`(rtsps?://[^:/@\s]*:)[^/@\s]*@`)

// redact masks passwords in any URL FFmpeg echoes back.
func redact(line string) string {
	return credentials.ReplaceAllString(line, "${1}xxxxx@")
}

const maxDetailLength = 200

// linePrefix matches what FFmpeg puts in front of a message: the component
// tag ("[in#0/rtsp @ 0x55d0c]") and the input URL, which the client
// already shows next to the stream.
var linePrefix = regexp.MustCompile(`^(\[[^\]]*\] )?(rtsps?://\S+?: )?`)

// errorFromOutput builds an Error from FFmpeg's stderr, preferring the most
// recent line that explains the failure over trailing decoder noise.
func errorFromOutput(stderr []byte, cause error) *Error {
	var fallback string
	lines := bytes.Split(bytes.TrimSpace(stderr), []byte("\n"))
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(string(lines[i]))
		if line == "" || strings.HasPrefix(line, "Last message repeated") {
			continue
		}
		if kind, ok := classify(line); ok {
			return &Error{Kind: kind, Detail: detail(line), Err: cause}
		}
		if fallback == "" {
			fallback = line
		}
	}
	return &Error{Kind: KindExited, Detail: detail(fallback), Err: cause}
}

func detail(line string) string {
	line = redact(linePrefix.ReplaceAllString(line, ""))
	if len(line) > maxDetailLength {
		line = line[:maxDetailLength]
	}
	return line
}
