package data

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
)

// MediaJob is a row in media_processing_jobs.
type MediaJob struct {
	ID            uuid.UUID `db:"id"`
	StorageID     uuid.UUID `db:"storage_id"`
	StorageFileID uuid.UUID `db:"storage_file_id"`
	Status        string    `db:"status"`
	Attempts      int       `db:"attempts"`
	LastError     sql.NullString `db:"last_error"`
	CreatedAt     time.Time `db:"created_at"`
	UpdatedAt     time.Time `db:"updated_at"`
}

// EnqueueMediaProcessing inserts a pending job or no-ops if one already exists for the file.
func (d *DB) EnqueueMediaProcessing(ctx context.Context, storageID, storageFileID uuid.UUID) error {
	id := uuid.New()
	_, err := d.db.ExecContext(ctx, `
		INSERT INTO media_processing_jobs (id, storage_id, storage_file_id, status)
		VALUES ($1, $2, $3, 'pending')
		ON CONFLICT (storage_file_id) DO NOTHING`,
		id, storageID, storageFileID)
	return err
}

// ClaimMediaJob locks and returns the next pending job, or nil if the queue is empty.
func (d *DB) ClaimMediaJob(ctx context.Context) (*MediaJob, error) {
	var j MediaJob
	err := d.db.GetContext(ctx, &j, `
		WITH c AS (
			SELECT id FROM media_processing_jobs
			WHERE status = 'pending'
			ORDER BY created_at ASC
			FOR UPDATE SKIP LOCKED
			LIMIT 1
		)
		UPDATE media_processing_jobs j
		SET status = 'processing', updated_at = NOW(), attempts = j.attempts + 1
		FROM c WHERE j.id = c.id
		RETURNING j.id, j.storage_id, j.storage_file_id, j.status, j.attempts, j.last_error, j.created_at, j.updated_at`)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &j, nil
}

// CompleteMediaJob marks a job finished successfully.
func (d *DB) CompleteMediaJob(ctx context.Context, jobID uuid.UUID) error {
	_, err := d.db.ExecContext(ctx, `
		UPDATE media_processing_jobs SET status = 'done', last_error = NULL, updated_at = NOW() WHERE id = $1`,
		jobID)
	return err
}

// FailMediaJob records a terminal failure for this attempt.
func (d *DB) FailMediaJob(ctx context.Context, jobID uuid.UUID, msg string) error {
	if len(msg) > 8000 {
		msg = msg[:8000]
	}
	_, err := d.db.ExecContext(ctx, `
		UPDATE media_processing_jobs SET status = 'failed', last_error = $2, updated_at = NOW() WHERE id = $1`,
		jobID, msg)
	return err
}
