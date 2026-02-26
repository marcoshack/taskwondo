CREATE TABLE email_verification_tokens (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email        TEXT NOT NULL,
    display_name TEXT NOT NULL,
    token_hash   TEXT NOT NULL UNIQUE,
    expires_at   TIMESTAMPTZ NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_evt_email ON email_verification_tokens (email);
