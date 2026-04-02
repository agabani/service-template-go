CREATE TABLE users (
    id         BIGSERIAL   PRIMARY KEY,
    uuid       UUID        NOT NULL,
    email      TEXT        NOT NULL,
    name       TEXT        NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_users_uuid  ON users (uuid);
CREATE UNIQUE INDEX idx_users_email ON users (email);
