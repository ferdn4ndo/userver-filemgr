package media_processor

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func ffprobeBinary(t *testing.T) string {
	t.Helper()
	path, err := exec.LookPath("ffprobe")
	if err != nil {
		t.Skip("ffprobe not available (install ffmpeg for this test)")
	}
	return path
}

func TestFFprobeMeta_InvalidFile(t *testing.T) {
	bin := ffprobeBinary(t)
	ctx := context.Background()
	f := filepath.Join(t.TempDir(), "not-a-video.bin")
	require.NoError(t, os.WriteFile(f, []byte("hello"), 0o600))

	dur, w, h, fps := ffprobeMeta(ctx, bin, f)
	assert.Zero(t, dur)
	assert.Zero(t, w)
	assert.Zero(t, h)
	assert.False(t, fps.Valid)
}
