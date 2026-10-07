-- OAuth identities (option B: password auth stays, GitHub/Discord added).
ALTER TABLE users ALTER COLUMN password_hash DROP NOT NULL;

CREATE TABLE oauth_identities (
    provider       text        NOT NULL CHECK (provider IN ('github', 'discord')),
    provider_uid   text        NOT NULL,
    user_id        uuid        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at     timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (provider, provider_uid)
);
CREATE INDEX idx_oauth_identities_user_id ON oauth_identities(user_id);
