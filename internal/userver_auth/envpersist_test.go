package userver_auth

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormatEnvValue(t *testing.T) {
	assert.Equal(t, "plain", formatEnvValue("plain"))
	assert.Equal(t, `"a=b"`, formatEnvValue("a=b"))
	assert.Equal(t, `"say \"hi\""`, formatEnvValue(`say "hi"`))
	assert.Equal(t, `p\q`, formatEnvValue(`p\q`))
	assert.Equal(t, `"line1\nline2"`, formatEnvValue("line1\nline2"))
	assert.Equal(t, `""`, formatEnvValue(""))
}

func TestUpsertEnvFile_preserveBlankLines(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, ".env")
	require.NoError(t, os.WriteFile(p, []byte("FOO=1\n\nBAR=2\n"), 0o600))

	require.NoError(t, upsertEnvFile(p, map[string]string{
		"FILEMGR_BOOTSTRAP_ADMIN_USERNAME": "u",
	}))

	raw, err := os.ReadFile(p)
	require.NoError(t, err)
	s := string(raw)
	assert.Contains(t, s, "FOO=1")
	assert.Contains(t, s, "BAR=2")
	assert.Contains(t, s, "FILEMGR_BOOTSTRAP_ADMIN_USERNAME=u")
}

func TestParseEnvValue_quoted(t *testing.T) {
	assert.Equal(t, `a"b`, parseEnvValue("\"a\\\"b\""))
	assert.Equal(t, "x\ny", parseEnvValue("\"x\\ny\""))
}

func TestFilterUpdatesSkipProtectedInFile(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, ".env")
	require.NoError(t, os.WriteFile(p, []byte("FILEMGR_BOOTSTRAP_ADMIN_USERNAME=keep_me\nFILEMGR_SYSTEM_TOKEN=<PLACEHOLDER>\n"), 0o600))

	out, err := filterUpdatesSkipProtectedInFile(p, map[string]string{
		"FILEMGR_BOOTSTRAP_ADMIN_USERNAME": "runtime_user",
		"FILEMGR_SYSTEM_TOKEN":             "new-token",
		"FILEMGR_BOOTSTRAP_ADMIN_PASSWORD": "newpass",
	})
	require.NoError(t, err)
	assert.NotContains(t, out, "FILEMGR_BOOTSTRAP_ADMIN_USERNAME")
	assert.Equal(t, "new-token", out["FILEMGR_SYSTEM_TOKEN"])
	assert.Equal(t, "newpass", out["FILEMGR_BOOTSTRAP_ADMIN_PASSWORD"])
}

func TestUpsertEnvFile_replacesDuplicateKeys(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, ".env")
	require.NoError(t, os.WriteFile(p, []byte("X=old\nFILEMGR_BOOTSTRAP_ADMIN_USERNAME=old1\nFILEMGR_BOOTSTRAP_ADMIN_USERNAME=old2\n"), 0o600))

	require.NoError(t, upsertEnvFile(p, map[string]string{
		"FILEMGR_BOOTSTRAP_ADMIN_USERNAME": "newuser",
	}))

	raw, err := os.ReadFile(p)
	require.NoError(t, err)
	c := string(raw)
	assert.Equal(t, 1, strings.Count(c, "FILEMGR_BOOTSTRAP_ADMIN_USERNAME="))
	assert.Contains(t, c, "FILEMGR_BOOTSTRAP_ADMIN_USERNAME=newuser")
}
