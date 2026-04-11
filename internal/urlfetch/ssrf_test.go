package urlfetch

import (
	"io"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateFetchURL_blocksPrivateIP(t *testing.T) {
	u, err := url.Parse("https://10.0.0.1/foo")
	require.NoError(t, err)
	err = validateFetchURL(u, false, false)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrURLNotAllowed)
}

func TestValidateFetchURL_allowsLoopbackNonProd(t *testing.T) {
	u, err := url.Parse("https://127.0.0.1/foo")
	require.NoError(t, err)
	err = validateFetchURL(u, false, true)
	assert.NoError(t, err)
}

func TestValidateFetchURL_httpRequiresFlag(t *testing.T) {
	u, err := url.Parse("http://example.com/foo")
	require.NoError(t, err)
	err = validateFetchURL(u, false, false)
	require.Error(t, err)
	err = validateFetchURL(u, true, false)
	assert.NoError(t, err)
}

func TestCappedReader_Max(t *testing.T) {
	cr := &CappedReader{R: strings.NewReader("abcd"), Max: 2}
	b, err := io.ReadAll(cr)
	require.Error(t, err)
	assert.Len(t, b, 2)
}
