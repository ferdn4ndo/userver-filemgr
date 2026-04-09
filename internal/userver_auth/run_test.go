package userver_auth

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ferdn4ndo/userver-filemgr/lib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRun_NoOpWhenUnset(t *testing.T) {
	t.Setenv("SKIP_AUTH_BOOTSTRAP", "")
	for _, k := range []string{"SYSTEM_CREATION_TOKEN", "FILEMGR_BOOTSTRAP_SYSTEM_NAME", "FILEMGR_BOOTSTRAP_ADMIN_USERNAME", "FILEMGR_BOOTSTRAP_ADMIN_PASSWORD", "FILEMGR_SYSTEM_TOKEN", "AUTH_HOST", "USERVER_AUTH_HOST"} {
		t.Setenv(k, "")
	}
	var buf bytes.Buffer
	env := lib.Env{AuthHost: "", AuthHTTPTimeout: 5 * time.Second}
	require.NoError(t, Run(&buf, env))
	assert.Contains(t, buf.String(), "nothing to do")
}

func TestRun_CreatesSystemAndRegisters(t *testing.T) {
	var sysTok string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/auth/system":
			assert.Equal(t, "Token creation-secret", r.Header.Get("Authorization"))
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			assert.Equal(t, "mysys", body["name"])
			sysTok = "generated-system-token"
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{"name": "mysys", "token": sysTok})
		case r.Method == http.MethodPost && r.URL.Path == "/auth/register":
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			assert.Equal(t, "admin", body["username"])
			assert.Equal(t, "mysys", body["system_name"])
			assert.Equal(t, sysTok, body["system_token"])
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)

	t.Setenv("SYSTEM_CREATION_TOKEN", "creation-secret")
	t.Setenv("FILEMGR_BOOTSTRAP_SYSTEM_NAME", "mysys")
	t.Setenv("FILEMGR_BOOTSTRAP_ADMIN_USERNAME", "admin")
	t.Setenv("FILEMGR_BOOTSTRAP_ADMIN_PASSWORD", "secret-pass")
	t.Setenv("SKIP_AUTH_BOOTSTRAP", "")

	envFile := filepath.Join(t.TempDir(), ".env")
	t.Setenv("ENV_FILE", envFile)
	t.Cleanup(func() { _ = os.Unsetenv("ENV_FILE") })

	var buf bytes.Buffer
	env := lib.Env{AuthHost: srv.URL, AuthHTTPTimeout: 5 * time.Second}
	require.NoError(t, Run(&buf, env))
	assert.Contains(t, buf.String(), "created system")
	assert.Contains(t, buf.String(), "registered user")
	assert.Contains(t, buf.String(), "persisted bootstrap fields to")
	assert.Contains(t, buf.String(), "were not overwritten")

	raw, err := os.ReadFile(envFile)
	require.NoError(t, err)
	s := string(raw)
	assert.Contains(t, s, "FILEMGR_SYSTEM_TOKEN=generated-system-token")
	assert.Contains(t, s, "FILEMGR_BOOTSTRAP_ADMIN_USERNAME=admin")
	assert.Contains(t, s, "FILEMGR_BOOTSTRAP_ADMIN_PASSWORD=secret-pass")
	assert.Contains(t, s, "FILEMGR_BOOTSTRAP_ADMIN_IS_ADMIN=1")
}

func TestRun_RegisterOnlyWithExistingSystem(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/auth/register" {
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			assert.Equal(t, "existing-sys", body["system_name"])
			assert.Equal(t, "known-token", body["system_token"])
			w.WriteHeader(http.StatusCreated)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)

	t.Setenv("SYSTEM_CREATION_TOKEN", "")
	t.Setenv("FILEMGR_BOOTSTRAP_SYSTEM_NAME", "existing-sys")
	t.Setenv("FILEMGR_SYSTEM_TOKEN", "known-token")
	t.Setenv("FILEMGR_BOOTSTRAP_ADMIN_USERNAME", "u1")
	t.Setenv("FILEMGR_BOOTSTRAP_ADMIN_PASSWORD", "p1")
	t.Setenv("SKIP_AUTH_BOOTSTRAP", "")

	envFile := filepath.Join(t.TempDir(), ".env")
	t.Setenv("ENV_FILE", envFile)
	t.Cleanup(func() { _ = os.Unsetenv("ENV_FILE") })

	var buf bytes.Buffer
	env := lib.Env{AuthHost: srv.URL, AuthHTTPTimeout: 5 * time.Second}
	require.NoError(t, Run(&buf, env))
	assert.Contains(t, buf.String(), "registered user")

	raw, err := os.ReadFile(envFile)
	require.NoError(t, err)
	s := string(raw)
	assert.Contains(t, s, "FILEMGR_SYSTEM_TOKEN=known-token")
	assert.Contains(t, s, "FILEMGR_BOOTSTRAP_ADMIN_USERNAME=u1")
	assert.Contains(t, strings.TrimSpace(s), "FILEMGR_BOOTSTRAP_ADMIN_PASSWORD=p1")
}

func TestRun_PersistPreservesExistingNonEmptyEnvFile(t *testing.T) {
	var sysTok string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/auth/system":
			sysTok = "generated-system-token"
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{"name": "mysys", "token": sysTok})
		case r.Method == http.MethodPost && r.URL.Path == "/auth/register":
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			assert.Equal(t, "api_user", body["username"])
			w.WriteHeader(http.StatusCreated)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)

	envFile := filepath.Join(t.TempDir(), ".env")
	require.NoError(t, os.WriteFile(envFile, []byte(
		"FILEMGR_BOOTSTRAP_ADMIN_USERNAME=disk_user\n"+
			"FILEMGR_BOOTSTRAP_ADMIN_PASSWORD=disk_pass\n"+
			"FILEMGR_BOOTSTRAP_ADMIN_IS_ADMIN=0\n",
	), 0o600))
	t.Setenv("ENV_FILE", envFile)
	t.Cleanup(func() { _ = os.Unsetenv("ENV_FILE") })

	t.Setenv("SYSTEM_CREATION_TOKEN", "creation-secret")
	t.Setenv("FILEMGR_BOOTSTRAP_SYSTEM_NAME", "mysys")
	t.Setenv("FILEMGR_BOOTSTRAP_ADMIN_USERNAME", "api_user")
	t.Setenv("FILEMGR_BOOTSTRAP_ADMIN_PASSWORD", "api_pass")
	t.Setenv("SKIP_AUTH_BOOTSTRAP", "")

	var buf bytes.Buffer
	env := lib.Env{AuthHost: srv.URL, AuthHTTPTimeout: 5 * time.Second}
	require.NoError(t, Run(&buf, env))

	raw, err := os.ReadFile(envFile)
	require.NoError(t, err)
	s := string(raw)
	assert.Contains(t, s, "FILEMGR_BOOTSTRAP_ADMIN_USERNAME=disk_user")
	assert.Contains(t, s, "FILEMGR_BOOTSTRAP_ADMIN_PASSWORD=disk_pass")
	assert.Contains(t, s, "FILEMGR_BOOTSTRAP_ADMIN_IS_ADMIN=0")
	assert.Contains(t, s, "FILEMGR_SYSTEM_TOKEN=generated-system-token")
}

func TestRun_SkipPersistBootstrapEnv(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/auth/system" {
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{"name": "mysys", "token": "tok"})
			return
		}
		if r.Method == http.MethodPost && r.URL.Path == "/auth/register" {
			w.WriteHeader(http.StatusCreated)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)

	t.Setenv("SYSTEM_CREATION_TOKEN", "creation-secret")
	t.Setenv("FILEMGR_BOOTSTRAP_SYSTEM_NAME", "mysys")
	t.Setenv("FILEMGR_BOOTSTRAP_ADMIN_USERNAME", "admin")
	t.Setenv("FILEMGR_BOOTSTRAP_ADMIN_PASSWORD", "secret-pass")
	t.Setenv("SKIP_AUTH_BOOTSTRAP", "")
	t.Setenv("FILEMGR_SKIP_PERSIST_BOOTSTRAP_ENV", "1")

	envFile := filepath.Join(t.TempDir(), ".env")
	t.Setenv("ENV_FILE", envFile)
	t.Cleanup(func() {
		_ = os.Unsetenv("ENV_FILE")
		_ = os.Unsetenv("FILEMGR_SKIP_PERSIST_BOOTSTRAP_ENV")
	})

	var buf bytes.Buffer
	env := lib.Env{AuthHost: srv.URL, AuthHTTPTimeout: 5 * time.Second}
	require.NoError(t, Run(&buf, env))
	assert.NotContains(t, buf.String(), "persisted FILEMGR_BOOTSTRAP_ADMIN_*")
	_, err := os.Stat(envFile)
	assert.True(t, os.IsNotExist(err))
}
