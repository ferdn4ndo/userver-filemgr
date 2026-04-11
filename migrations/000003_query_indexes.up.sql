-- Performance: indexes aligned with common list/detail queries (safe IF NOT EXISTS).

CREATE INDEX IF NOT EXISTS core_storagefile_storage_excluded_created_idx
    ON core_storagefile (storage_id, excluded, created_at DESC);

CREATE INDEX IF NOT EXISTS core_storagefiledownload_expires_idx
    ON core_storagefiledownload (expires_at);
