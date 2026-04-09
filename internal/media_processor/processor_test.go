package media_processor

import (
	"database/sql"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/ferdn4ndo/userver-filemgr/internal/data"
)

func TestMediaCategory(t *testing.T) {
	t.Parallel()
	id := uuid.New()
	require.Equal(t, "", MediaCategory(nil))
	require.Equal(t, "", MediaCategory(&data.StorageFile{Type: nil}))
	require.Equal(t, "IMAGE", MediaCategory(&data.StorageFile{
		Type: &data.MimeType{MimeType: "image/png", GenericType: sql.NullString{String: "IMAGE", Valid: true}},
	}))
	require.Equal(t, "VIDEO", MediaCategory(&data.StorageFile{
		Type: &data.MimeType{MimeType: "video/mp4"},
	}))
	require.Equal(t, "DOCUMENT", MediaCategory(&data.StorageFile{
		Type: &data.MimeType{MimeType: "application/pdf"},
	}))
	require.True(t, ShouldEnqueue(&data.StorageFile{
		ID: id, Type: &data.MimeType{MimeType: "image/jpeg"},
	}))
}
