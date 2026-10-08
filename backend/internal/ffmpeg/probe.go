package ffmpeg

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"slices"
)

// ProbeResult describes the first video stream of a source.
type ProbeResult struct {
	Codec   string `json:"codec_name"`
	Profile string `json:"profile"`
	PixFmt  string `json:"pix_fmt"`
}

var unplayableProfiles = []string{"High 10", "High 4:2:2", "High 4:4:4 Predictive"}

// Mode picks remuxing when browsers can decode the source as-is. MSE support
// for H.264 is limited to 8-bit 4:2:0, so other profiles are re-encoded.
func (p ProbeResult) Mode() Mode {
	if p.Codec != "h264" {
		return ModeTranscode
	}
	if p.PixFmt != "" && p.PixFmt != "yuv420p" && p.PixFmt != "yuvj420p" {
		return ModeTranscode
	}
	if slices.Contains(unplayableProfiles, p.Profile) {
		return ModeTranscode
	}
	return ModeCopy
}

func Probe(ctx context.Context, ffprobePath, url string) (ProbeResult, error) {
	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, ffprobePath, ProbeArgs(url)...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return ProbeResult{}, &Error{Kind: KindUnreachable, Detail: "Timed out waiting for the camera", Err: ctx.Err()}
		}
		return ProbeResult{}, errorFromOutput(stderr.Bytes(), err)
	}

	var out struct {
		Streams []ProbeResult `json:"streams"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
		return ProbeResult{}, fmt.Errorf("decode ffprobe output: %w", err)
	}
	if len(out.Streams) == 0 {
		return ProbeResult{}, &Error{Kind: KindNoVideo, Detail: "Source has no video stream", Err: errors.New("no video stream")}
	}
	return out.Streams[0], nil
}
