package data

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetMimeByMIME_sqlmock(t *testing.T) {
	sqldb, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqldb.Close() })
	db := &DB{db: sqlx.NewDb(sqldb, "postgres")}

	mid := uuid.MustParse("660e8400-e29b-41d4-a716-446655440010")
	rows := sqlmock.NewRows([]string{"id", "mime_type", "generic_type", "description", "extensions"}).
		AddRow(mid, "image/jpeg", "IMAGE", nil, "jpg,jpeg")

	mock.ExpectQuery(`FROM core_storagefilemimetype`).
		WithArgs("image/jpeg").
		WillReturnRows(rows)

	mt, err := db.GetMimeByMIME(context.Background(), "image/jpeg")
	require.NoError(t, err)
	assert.Equal(t, mid, mt.ID)
	assert.Equal(t, "image/jpeg", mt.MimeType)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestStorageMediaExistsForFile_sqlmock(t *testing.T) {
	sqldb, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqldb.Close() })
	db := &DB{db: sqlx.NewDb(sqldb, "postgres")}

	fid := uuid.MustParse("770e8400-e29b-41d4-a716-446655440011")
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM core_storagemedia WHERE storage_file_id`).
		WithArgs(fid).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	ok, err := db.StorageMediaExistsForFile(context.Background(), fid)
	require.NoError(t, err)
	assert.True(t, ok)
	require.NoError(t, mock.ExpectationsWereMet())
}
