CREATE TABLE gopherkeeper.records (
    id UUID PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES gopherkeeper.users(id) ON DELETE CASCADE,
    type TEXT NOT NULL,
    title TEXT NOT NULL,
    revision BIGINT NOT NULL DEFAULT 1,
    crypto_version SMALLINT NOT NULL,
    key_id TEXT NOT NULL,
    nonce BYTEA NOT NULL,
    ciphertext BYTEA NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT records_type_check CHECK (type IN ('text', 'credentials', 'card', 'binary')),
    CONSTRAINT records_title_not_empty CHECK (length(title) > 0),
    CONSTRAINT records_revision_positive CHECK (revision > 0),
    CONSTRAINT records_crypto_version_positive CHECK (crypto_version > 0),
    CONSTRAINT records_key_id_not_empty CHECK (length(key_id) > 0),
    CONSTRAINT records_nonce_not_empty CHECK (octet_length(nonce) > 0),
    CONSTRAINT records_ciphertext_not_empty CHECK (octet_length(ciphertext) > 0)
);

CREATE INDEX records_user_id_updated_at_idx ON gopherkeeper.records (user_id, updated_at DESC, id DESC);
