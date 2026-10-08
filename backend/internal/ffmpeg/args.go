package ffmpeg

import (
	"slices"
	"strconv"
	"time"
)

type Mode int

const (
	ModeCopy Mode = iota
	ModeTranscode
)

func (m Mode) String() string {
	if m == ModeTranscode {
		return "transcode"
	}
	return "copy"
}

const (
	// socketTimeout is FFmpeg's RTSP socket I/O timeout (-timeout, in
	// microseconds since FFmpeg 5). It turns a dead camera into an error
	// instead of a hang.
	socketTimeout = 5 * time.Second
)

// inputArgs keep probing short: the SDP already carries the codec
// parameters, so there is no need for FFmpeg's default 5s analysis window.
func inputArgs(url string) []string {
	return []string{
		"-rtsp_transport", "tcp",
		"-timeout", micros(socketTimeout),
		"-analyzeduration", "500000",
		"-probesize", "500000",
		"-i", url,
	}
}

func ProbeArgs(url string) []string {
	return slices.Concat(
		[]string{"-v", "error"},
		inputArgs(url),
		[]string{
			"-select_streams", "v:0",
			"-show_entries", "stream=codec_name,profile,pix_fmt",
			"-of", "json",
		},
	)
}

func StreamArgs(url string, mode Mode) []string {
	video := []string{"-c:v", "copy"}
	if mode == ModeTranscode {
		video = []string{
			"-c:v", "libx264",
			"-preset", "veryfast",
			"-tune", "zerolatency",
			"-profile:v", "high",
			"-pix_fmt", "yuv420p",
			"-vf", "scale=-2:'min(720,ih)'",
			"-force_key_frames", "expr:gte(t,n_forced*1)",
			"-crf", "23",
			"-maxrate", "2500k",
			"-bufsize", "5000k",
		}
	}

	// frag_keyframe starts a new fragment at every keyframe, so each fragment
	// can be decoded on its own and a new viewer can start from any of them.
	return slices.Concat(
		[]string{"-hide_banner", "-nostdin", "-loglevel", "warning", "-fflags", "+genpts+nobuffer"},
		inputArgs(url),
		[]string{"-map", "0:v:0", "-an", "-sn", "-dn"},
		video,
		[]string{
			"-f", "mp4",
			"-movflags", "frag_keyframe+empty_moov+default_base_moof",
			"-flush_packets", "1",
			"pipe:1",
		},
	)
}

func micros(d time.Duration) string {
	return strconv.FormatInt(d.Microseconds(), 10)
}
