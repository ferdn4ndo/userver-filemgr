//go:build integration

package media_processor

import (
	"context"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestFFprobeMeta_MinimalMP4 encodes a 1-frame clip; run with: go test -tags=integration ./internal/media_processor
func TestFFprobeMeta_MinimalMP4(t *testing.T) {
	ffprobe := ffprobeBinary(t)
	ffmpeg, err := exec.LookPath("ffmpeg")
	if err != nil {
		t.Skip("ffmpeg not available")
	}
	out := filepath.Join(t.TempDir(), "oneframe.mp4")
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, ffmpeg,
		"-y", "-f", "lavfi", "-i", "color=c=black:s=320x240:r=1",
		"-frames:v", "1", "-c:v", "libx264", "-preset", "ultrafast", "-pix_fmt", "yuv420p",
		"-movflags", "+faststart",
		out,
	)
	if outBytes, err := cmd.CombinedOutput(); err != nil {
		t.Skipf("ffmpeg encode failed: %v: %s", err, outBytes)
	}

	dur, w, h, fps := ffprobeMeta(context.Background(), ffprobe, out)
	assert.Equal(t, 320, w)
	assert.Equal(t, 240, h)
	assert.True(t, fps.Valid)
	assert.Greater(t, fps.Int64, int64(0))
	_ = dur
}
