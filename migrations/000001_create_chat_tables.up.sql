CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS rooms (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    room_id VARCHAR(128) NOT NULL UNIQUE,
    owner_identifier_hash CHAR(64) NOT NULL,
    is_destroyed BOOLEAN NOT NULL DEFAULT FALSE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_rooms_destroyed_expires_at
    ON rooms (is_destroyed, expires_at);

CREATE TABLE IF NOT EXISTS room_members (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    room_id VARCHAR(128) NOT NULL REFERENCES rooms(room_id) ON DELETE CASCADE,
    identifier_hash CHAR(64) NOT NULL,
    joined_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    left_at TIMESTAMPTZ,
    CONSTRAINT uq_room_members_room_identifier UNIQUE (room_id, identifier_hash)
);

CREATE INDEX IF NOT EXISTS idx_room_members_room_id
    ON room_members (room_id);

CREATE TABLE IF NOT EXISTS client_aliases (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    room_id VARCHAR(128) NOT NULL REFERENCES rooms(room_id) ON DELETE CASCADE,
    identifier_hash CHAR(64) NOT NULL,
    alias VARCHAR(128) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_client_aliases_room_identifier UNIQUE (room_id, identifier_hash)
);

CREATE INDEX IF NOT EXISTS idx_client_aliases_room_id
    ON client_aliases (room_id);

CREATE TABLE IF NOT EXISTS messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    room_id VARCHAR(128) NOT NULL REFERENCES rooms(room_id) ON DELETE CASCADE,
    body TEXT NOT NULL,
    sender_identifier_hash CHAR(64) NOT NULL,
    sender_alias VARCHAR(128) NOT NULL,
    sent_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_messages_room_sent_at
    ON messages (room_id, sent_at ASC);
