package data

import (
	"context"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetStorage_sqlmock(t *testing.T) {
	sqldb, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqldb.Close() })
	db := &DB{db: sqlx.NewDb(sqldb, "postgres")}

	id := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	rows := sqlmock.NewRows([]string{"id", "type", "credentials", "media_convert_configuration", "total_size"}).
		AddRow(id, "LOCAL", []byte("{}"), []byte("null"), int64(42))

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT s.id, s.type, s.credentials, s.media_convert_configuration,
			COALESCE((SELECT SUM(size) FROM core_storagefile f WHERE f.storage_id = s.id AND f.excluded = false), 0)::bigint AS total_size
		FROM core_storage s WHERE s.id = $1`)).
		WithArgs(id).
		WillReturnRows(rows)

	st, err := db.GetStorage(context.Background(), id)
	require.NoError(t, err)
	assert.Equal(t, id, st.ID)
	assert.Equal(t, int64(42), st.TotalSize)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestFindMimeByExtension_sqlmock(t *testing.T) {
	sqldb, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqldb.Close() })
	db := &DB{db: sqlx.NewDb(sqldb, "postgres")}
	mid := uuid.MustParse("660e8400-e29b-41d4-a716-446655440001")
	rows := sqlmock.NewRows([]string{"id", "mime_type", "generic_type", "description", "extensions"}).
		AddRow(mid, "image/jpeg", "IMAGE", nil, "jpg,jpeg")

	mock.ExpectQuery(`SELECT id, mime_type`).WillReturnRows(rows)
	mt, err := db.FindMimeByExtension(context.Background(), "jpg")
	require.NoError(t, err)
	assert.Equal(t, "image/jpeg", mt.MimeType)
	assert.NoError(t, mock.ExpectationsWereMet())
}
