package object_store

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ferdn4ndo/userver-filemgr/lib"
)

func TestLocalBackendPutAndAbs(t *testing.T) {
	root := t.TempDir()
	l := &localBackend{root: root}
	p := "a/b/c.txt"
	err := l.Put(context.Background(), p, strings.NewReader("hello"), 5, "text/plain")
	require.NoError(t, err)
	full := l.abs(p)
	b, err := os.ReadFile(full)
	require.NoError(t, err)
	assert.Equal(t, "hello", string(b))
	rc, err := l.Open(context.Background(), p)
	require.NoError(t, err)
	t.Cleanup(func() { _ = rc.Close() })
	b2, err := io.ReadAll(rc)
	require.NoError(t, err)
	assert.Equal(t, "hello", string(b2))
}

func TestFactoryForStorageLocal(t *testing.T) {
	env := lib.Env{LocalRoot: t.TempDir()}
	f := NewFactory(env)
	be, err := f.ForStorage("LOCAL", []byte(`{"LOCAL_ROOT": ""}`))
	require.NoError(t, err)
	err = be.Put(context.Background(), "k", strings.NewReader("x"), 1, "")
	assert.NoError(t, err)
	err = be.Put(context.Background(), "k2", strings.NewReader("yz"), -1, "")
	assert.NoError(t, err)
}

func TestFactoryForStorageUnknown(t *testing.T) {
	f := NewFactory(lib.Env{})
	_, err := f.ForStorage("OTHER", []byte(`{}`))
	assert.Error(t, err)
}

func TestLocalBackendAbs(t *testing.T) {
	l := &localBackend{root: "/r"}
	assert.True(t, strings.HasSuffix(l.abs("/x/y"), filepath.Join("x", "y")))
}
