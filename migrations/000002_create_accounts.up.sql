CREATE TABLE accounts (
    id         BIGSERIAL   PRIMARY KEY,
    uuid       UUID        NOT NULL,
    user_id    UUID        NOT NULL REFERENCES users (uuid) ON DELETE CASCADE,
    name       TEXT        NOT NULL,
    balance    BIGINT      NOT NULL DEFAULT 0,
    currency   CHAR(3)     NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_accounts_uuid    ON accounts (uuid);
CREATE        INDEX idx_accounts_user_id ON accounts (user_id);
