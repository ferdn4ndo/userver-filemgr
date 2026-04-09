-- Durable queue for async media processing (images, optional video metadata).
-- Multiple app instances can run workers; PostgreSQL SKIP LOCKED avoids double work.

CREATE TABLE IF NOT EXISTS media_processing_jobs (
    id UUID PRIMARY KEY,
    storage_id UUID NOT NULL REFERENCES core_storage(id) ON DELETE CASCADE,
    storage_file_id UUID NOT NULL REFERENCES core_storagefile(id) ON DELETE CASCADE,
    status VARCHAR(32) NOT NULL DEFAULT 'pending',
    attempts INT NOT NULL DEFAULT 0,
    last_error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS media_processing_jobs_storage_file_id_key ON media_processing_jobs(storage_file_id);
CREATE INDEX IF NOT EXISTS media_processing_jobs_status_created_idx ON media_processing_jobs(status, created_at);
