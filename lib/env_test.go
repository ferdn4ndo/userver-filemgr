package lib

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewEnv(t *testing.T) {
	t.Setenv("POSTGRES_HOST", "db")
	t.Setenv("POSTGRES_USER", "u")
	t.Setenv("POSTGRES_PASS", "p")
	t.Setenv("POSTGRES_DB", "app")
	t.Setenv("APP_PORT", "6000")
	t.Setenv("ENV_MODE", "prod")
	t.Setenv("DOWNLOAD_EXP_BYTES_SECS_RATIO", "4.5")
	t.Setenv("USERVER_AUTH_HOST", "http://auth/")
	t.Setenv("LOCAL_STORAGE_ROOT", "/data")
	t.Setenv("APP_PUBLIC_BASE_URL", "https://example.com")

	e := NewEnv()
	assert.Equal(t, "6000", e.ServerPort)
	assert.True(t, e.IsProduction())
	assert.False(t, e.IsLocal())
	assert.Equal(t, 4.5, e.DownloadRatio)
	assert.Equal(t, "http://auth", e.AuthHost)
	assert.Equal(t, "/data", e.LocalRoot)
	assert.Equal(t, "https://example.com", e.PublicBaseURL)
}

func TestNewEnvDefaults(t *testing.T) {
	t.Setenv("POSTGRES_HOST", "x")
	t.Setenv("POSTGRES_USER", "x")
	t.Setenv("POSTGRES_PASS", "x")
	t.Setenv("POSTGRES_DB", "x")
	for _, k := range []string{"APP_PORT", "ENV_MODE", "DOWNLOAD_EXP_BYTES_SECS_RATIO", "USERVER_AUTH_HOST", "AUTH_HOST", "FFPROBE_PATH"} {
		t.Setenv(k, "")
	}
	e := NewEnv()
	assert.Equal(t, "5000", e.ServerPort)
	assert.False(t, e.IsProduction())
	assert.Equal(t, 4.25, e.DownloadRatio)
}

func TestNewEnv_FFprobePathExplicit(t *testing.T) {
	t.Setenv("POSTGRES_HOST", "x")
	t.Setenv("POSTGRES_USER", "x")
	t.Setenv("POSTGRES_PASS", "x")
	t.Setenv("POSTGRES_DB", "x")
	t.Setenv("FFPROBE_PATH", "/custom/bin/ffprobe")
	e := NewEnv()
	assert.Equal(t, "/custom/bin/ffprobe", e.FFprobePath)
}

func TestNewEnv_AuthHostAlias(t *testing.T) {
	t.Setenv("POSTGRES_HOST", "x")
	t.Setenv("POSTGRES_USER", "x")
	t.Setenv("POSTGRES_PASS", "x")
	t.Setenv("POSTGRES_DB", "x")
	t.Setenv("USERVER_AUTH_HOST", "")
	t.Setenv("AUTH_HOST", "http://auth-service:5000/")
	e := NewEnv()
	assert.Equal(t, "http://auth-service:5000", e.AuthHost)
	assert.Equal(t, 15*time.Second, e.AuthHTTPTimeout)
}
