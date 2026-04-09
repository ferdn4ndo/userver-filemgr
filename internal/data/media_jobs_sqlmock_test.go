package data

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnqueueMediaProcessing_sqlmock(t *testing.T) {
	sqldb, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqldb.Close() })
	db := &DB{db: sqlx.NewDb(sqldb, "postgres")}

	sid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	fid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440001")

	mock.ExpectExec(`INSERT INTO media_processing_jobs`).
		WithArgs(sqlmock.AnyArg(), sid, fid).
		WillReturnResult(sqlmock.NewResult(0, 1))

	require.NoError(t, db.EnqueueMediaProcessing(context.Background(), sid, fid))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestClaimMediaJob_sqlmock_EmptyQueue(t *testing.T) {
	sqldb, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqldb.Close() })
	db := &DB{db: sqlx.NewDb(sqldb, "postgres")}

	mock.ExpectQuery(`WITH c AS`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "storage_id", "storage_file_id", "status", "attempts", "last_error", "created_at", "updated_at",
		}))

	job, err := db.ClaimMediaJob(context.Background())
	require.NoError(t, err)
	assert.Nil(t, job)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestClaimMediaJob_sqlmock_ClaimsRow(t *testing.T) {
	sqldb, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqldb.Close() })
	db := &DB{db: sqlx.NewDb(sqldb, "postgres")}

	jid := uuid.MustParse("650e8400-e29b-41d4-a716-446655440002")
	sid := uuid.MustParse("650e8400-e29b-41d4-a716-446655440003")
	fid := uuid.MustParse("650e8400-e29b-41d4-a716-446655440004")
	now := time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC)

	mock.ExpectQuery(`WITH c AS`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "storage_id", "storage_file_id", "status", "attempts", "last_error", "created_at", "updated_at",
		}).AddRow(jid, sid, fid, "processing", 1, nil, now, now))

	job, err := db.ClaimMediaJob(context.Background())
	require.NoError(t, err)
	require.NotNil(t, job)
	assert.Equal(t, jid, job.ID)
	assert.Equal(t, sid, job.StorageID)
	assert.Equal(t, fid, job.StorageFileID)
	assert.Equal(t, "processing", job.Status)
	assert.Equal(t, 1, job.Attempts)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCompleteMediaJob_sqlmock(t *testing.T) {
	sqldb, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqldb.Close() })
	db := &DB{db: sqlx.NewDb(sqldb, "postgres")}

	jid := uuid.MustParse("750e8400-e29b-41d4-a716-446655440005")
	mock.ExpectExec(`UPDATE media_processing_jobs SET status = 'done'`).
		WithArgs(jid).
		WillReturnResult(sqlmock.NewResult(0, 1))

	require.NoError(t, db.CompleteMediaJob(context.Background(), jid))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestFailMediaJob_sqlmock_TruncatesMessage(t *testing.T) {
	sqldb, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqldb.Close() })
	db := &DB{db: sqlx.NewDb(sqldb, "postgres")}

	jid := uuid.MustParse("750e8400-e29b-41d4-a716-446655440006")
	long := make([]byte, 9000)
	for i := range long {
		long[i] = 'x'
	}
	mock.ExpectExec(`UPDATE media_processing_jobs SET status = 'failed'`).
		WithArgs(jid, string(long[:8000])).
		WillReturnResult(sqlmock.NewResult(0, 1))

	require.NoError(t, db.FailMediaJob(context.Background(), jid, string(long)))
	require.NoError(t, mock.ExpectationsWereMet())
}
