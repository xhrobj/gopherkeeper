CREATE TABLE gopherkeeper.users (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    login TEXT NOT NULL,
    password_hash BYTEA NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT users_login_length CHECK (char_length(login) BETWEEN 3 AND 32),
    CONSTRAINT users_password_hash_not_empty CHECK (octet_length(password_hash) > 0),
    CONSTRAINT users_login_unique UNIQUE (login)
);
