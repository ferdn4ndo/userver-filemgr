-- Schema compatible with legacy Django ORM table names (core_*).
-- Safe on existing deployments: CREATE IF NOT EXISTS skips existing objects.
-- Greenfield: creates all tables needed by the Go service.

CREATE TABLE IF NOT EXISTS core_customuser (
    password VARCHAR(128) NOT NULL,
    last_login TIMESTAMPTZ NULL,
    is_superuser BOOLEAN NOT NULL,
    id UUID PRIMARY KEY,
    username VARCHAR(32) NOT NULL,
    system_name VARCHAR(255) NOT NULL,
    is_admin BOOLEAN NOT NULL,
    is_active BOOLEAN NOT NULL,
    registered_at TIMESTAMPTZ NOT NULL,
    last_activity_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS core_usertoken (
    uuid UUID PRIMARY KEY,
    token VARCHAR(255) NOT NULL UNIQUE,
    issued_at TIMESTAMPTZ NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    user_id UUID NOT NULL REFERENCES core_customuser(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS core_usertoken_user_id_idx ON core_usertoken(user_id);

CREATE TABLE IF NOT EXISTS core_storage (
    id UUID PRIMARY KEY,
    type VARCHAR(10) NULL,
    credentials JSONB NULL,
    media_convert_configuration JSONB NULL
);

CREATE TABLE IF NOT EXISTS core_storagefilemimetype (
    id UUID PRIMARY KEY,
    mime_type VARCHAR(255) NOT NULL UNIQUE,
    generic_type VARCHAR(50) NULL,
    description VARCHAR(255) NULL,
    extensions VARCHAR(255) NULL
);

CREATE TABLE IF NOT EXISTS core_storagefile (
    id UUID PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NULL,
    signature_key VARCHAR(64) NOT NULL,
    name VARCHAR(255) NULL,
    status VARCHAR(50) NOT NULL,
    visibility VARCHAR(50) NOT NULL,
    size BIGINT NOT NULL DEFAULT 0,
    hash VARCHAR(255) NULL,
    extension VARCHAR(16) NULL,
    exif_metadata JSONB NULL,
    custom_metadata JSONB NULL,
    origin VARCHAR(255) NOT NULL,
    original_path VARCHAR(1024) NULL,
    real_path VARCHAR(1024) NULL,
    virtual_path VARCHAR(1024) NULL,
    available BOOLEAN NOT NULL DEFAULT TRUE,
    excluded BOOLEAN NOT NULL DEFAULT FALSE,
    storage_id UUID NOT NULL REFERENCES core_storage(id) ON DELETE CASCADE,
    owner_id UUID NULL REFERENCES core_customuser(id) ON DELETE SET NULL,
    created_by_id UUID NULL REFERENCES core_customuser(id) ON DELETE SET NULL,
    updated_by_id UUID NULL REFERENCES core_customuser(id) ON DELETE SET NULL,
    type_id UUID NULL REFERENCES core_storagefilemimetype(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS core_storagefile_storage_excluded_idx ON core_storagefile(storage_id, excluded);

CREATE TABLE IF NOT EXISTS core_storageuser (
    id UUID PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NULL,
    may_write BOOLEAN NOT NULL DEFAULT FALSE,
    may_read BOOLEAN NOT NULL DEFAULT TRUE,
    storage_id UUID NOT NULL REFERENCES core_storage(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES core_customuser(id) ON DELETE CASCADE,
    created_by_id UUID NULL REFERENCES core_customuser(id) ON DELETE SET NULL,
    updated_by_id UUID NULL REFERENCES core_customuser(id) ON DELETE SET NULL,
    UNIQUE(storage_id, user_id)
);

CREATE TABLE IF NOT EXISTS core_storagefiledownload (
    id UUID PRIMARY KEY,
    download_url VARCHAR(1024) NULL,
    force_download BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    owner_id UUID NULL REFERENCES core_customuser(id) ON DELETE SET NULL,
    storage_file_id UUID NOT NULL REFERENCES core_storagefile(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS core_storagefiledownload_owner_file_idx ON core_storagefiledownload(owner_id, storage_file_id);

CREATE TABLE IF NOT EXISTS core_storagemedia (
    id UUID PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NULL,
    title VARCHAR(255) NULL,
    type VARCHAR(64) NOT NULL,
    description TEXT NULL,
    storage_file_id UUID NOT NULL UNIQUE REFERENCES core_storagefile(id) ON DELETE CASCADE,
    created_by_id UUID NULL REFERENCES core_customuser(id) ON DELETE SET NULL,
    updated_by_id UUID NULL REFERENCES core_customuser(id) ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS core_storagemediaimage (
    id UUID PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NULL,
    focal_length DOUBLE PRECISION NULL,
    aperture VARCHAR(255) NULL,
    flash_fired BOOLEAN NULL,
    iso INTEGER NULL,
    orientation_angle INTEGER NULL,
    is_flipped BOOLEAN NULL,
    exposition VARCHAR(255) NULL,
    datetime_taken VARCHAR(255) NULL,
    camera_manufacturer VARCHAR(255) NULL,
    camera_model VARCHAR(255) NULL,
    exif_image_height INTEGER NULL,
    exif_image_width INTEGER NULL,
    size_tag VARCHAR(64) NULL,
    height INTEGER NULL,
    width INTEGER NULL,
    megapixels NUMERIC(7,4) NULL,
    media_id UUID NOT NULL UNIQUE REFERENCES core_storagemedia(id) ON DELETE CASCADE,
    created_by_id UUID NULL REFERENCES core_customuser(id) ON DELETE SET NULL,
    updated_by_id UUID NULL REFERENCES core_customuser(id) ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS core_storagemediavideo (
    id UUID PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NULL,
    fps INTEGER NULL,
    duration INTERVAL NULL,
    size_tag VARCHAR(64) NULL,
    height INTEGER NULL,
    width INTEGER NULL,
    megapixels NUMERIC(7,4) NULL,
    media_id UUID NOT NULL UNIQUE REFERENCES core_storagemedia(id) ON DELETE CASCADE,
    created_by_id UUID NULL REFERENCES core_customuser(id) ON DELETE SET NULL,
    updated_by_id UUID NULL REFERENCES core_customuser(id) ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS core_storagemediathumbnail (
    id UUID PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NULL,
    size_tag VARCHAR(64) NULL,
    height INTEGER NULL,
    width INTEGER NULL,
    megapixels NUMERIC(7,4) NULL,
    media_id UUID NOT NULL REFERENCES core_storagemedia(id) ON DELETE CASCADE,
    storage_file_id UUID NOT NULL REFERENCES core_storagefile(id) ON DELETE CASCADE,
    created_by_id UUID NULL REFERENCES core_customuser(id) ON DELETE SET NULL,
    updated_by_id UUID NULL REFERENCES core_customuser(id) ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS core_storagemediaimagesized (
    id UUID PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NULL,
    size_tag VARCHAR(64) NULL,
    height INTEGER NULL,
    width INTEGER NULL,
    megapixels NUMERIC(7,4) NULL,
    media_image_id UUID NOT NULL REFERENCES core_storagemediaimage(id) ON DELETE CASCADE,
    storage_file_id UUID NOT NULL UNIQUE REFERENCES core_storagefile(id) ON DELETE CASCADE,
    created_by_id UUID NULL REFERENCES core_customuser(id) ON DELETE SET NULL,
    updated_by_id UUID NULL REFERENCES core_customuser(id) ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS core_storagemediadocument (
    id UUID PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NULL,
    pages INTEGER NULL,
    black_and_white BOOLEAN NOT NULL DEFAULT FALSE,
    media_id UUID NOT NULL UNIQUE REFERENCES core_storagemedia(id) ON DELETE CASCADE,
    created_by_id UUID NULL REFERENCES core_customuser(id) ON DELETE SET NULL,
    updated_by_id UUID NULL REFERENCES core_customuser(id) ON DELETE SET NULL
);
