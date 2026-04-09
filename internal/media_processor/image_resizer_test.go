package media_processor

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseImageResizerSizes_empty(t *testing.T) {
	b, err := parseImageResizerSizes(nil)
	require.NoError(t, err)
	assert.Nil(t, b)
	b, err = parseImageResizerSizes(json.RawMessage(`{}`))
	require.NoError(t, err)
	assert.Nil(t, b)
}

func TestParseImageResizerSizes_stringSizes(t *testing.T) {
	cfg := json.RawMessage(`{"image_resizer":{"sizes":["1920x1080","640x480 "]}}`)
	b, err := parseImageResizerSizes(cfg)
	require.NoError(t, err)
	assert.Equal(t, []imageBox{{1920, 1080}, {640, 480}}, b)
}

func TestParseImageResizerSizes_objectSizes(t *testing.T) {
	cfg := json.RawMessage(`{"image_resizer":{"sizes":[{"width":1280,"height":720}]}}`)
	b, err := parseImageResizerSizes(cfg)
	require.NoError(t, err)
	assert.Equal(t, []imageBox{{1280, 720}}, b)
}

func TestComputeNewImageDimensions_portrait(t *testing.T) {
	// Portrait: when height > width, factor = expectedW/origW (width becomes expectedW).
	rw, rh := computeNewImageDimensions(600, 1200, 1920, 1080)
	assert.Equal(t, 1920, rw)
	assert.Equal(t, 3840, rh)
}

func TestComputeNewImageDimensions_landscape(t *testing.T) {
	// Landscape: factor = expectedH/origH
	rw, rh := computeNewImageDimensions(2400, 800, 1920, 1080)
	assert.Equal(t, 3240, rw)
	assert.Equal(t, 1080, rh)
}

func TestSizeTagFromDimensions(t *testing.T) {
	assert.Equal(t, "SIZE_VGA", sizeTagFromDimensions(100, 100))
	assert.Equal(t, "SIZE_1K", sizeTagFromDimensions(1280, 864))
	assert.Equal(t, "SIZE_2K", sizeTagFromDimensions(2048, 1376))
}
