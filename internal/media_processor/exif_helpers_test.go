package media_processor

import (
	"database/sql"
	"testing"

	"github.com/ferdn4ndo/userver-filemgr/internal/data"
	"github.com/stretchr/testify/assert"
)

func TestFirstNonEmpty(t *testing.T) {
	t.Parallel()
	a := sql.NullString{String: "a", Valid: true}
	b := sql.NullString{String: "b", Valid: true}
	assert.Equal(t, a, firstNonEmpty(a, b))
	assert.Equal(t, b, firstNonEmpty(sql.NullString{}, b))
	assert.Equal(t, b, firstNonEmpty(sql.NullString{String: "", Valid: true}, b))
}

func TestFillEXIF_NilExif_SetsDimensions(t *testing.T) {
	t.Parallel()
	img := &data.MediaImage{}
	fillEXIF(img, nil, 640, 480)
	assert.Equal(t, int64(640), img.ExifImageWidth.Int64)
	assert.True(t, img.ExifImageWidth.Valid)
	assert.Equal(t, int64(480), img.ExifImageHeight.Int64)
	assert.True(t, img.ExifImageHeight.Valid)
}
