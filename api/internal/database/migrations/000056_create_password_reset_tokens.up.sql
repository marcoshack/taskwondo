CREATE TABLE password_reset_tokens (
    id         UUID PRIMARY KEY,
    email      TEXT NOT NULL,
    token_hash TEXT NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_password_reset_tokens_email ON password_reset_tokens (email);
