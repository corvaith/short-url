-- 0001_init.up.sql
CREATE TABLE users (
  id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  email         text NOT NULL,
  password_hash text NOT NULL,
  created_at    timestamptz NOT NULL DEFAULT now(),
  updated_at    timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT users_email_key   UNIQUE (email),
  CONSTRAINT users_email_len   CHECK (char_length(email) BETWEEN 3 AND 254),
  CONSTRAINT users_email_lower CHECK (email = lower(email))
);

CREATE TABLE sessions (
  id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id      uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  token_hash   bytea NOT NULL,
  created_at   timestamptz NOT NULL DEFAULT now(),
  expires_at   timestamptz NOT NULL,
  last_used_at timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT sessions_token_hash_key UNIQUE (token_hash)
);
CREATE INDEX sessions_user_id_idx    ON sessions (user_id);
CREATE INDEX sessions_expires_at_idx ON sessions (expires_at);

CREATE TABLE short_urls (
  id               uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id          uuid NULL REFERENCES users(id) ON DELETE CASCADE,
  code             text NOT NULL,
  target_url       text NOT NULL,
  clicks           bigint NOT NULL DEFAULT 0,
  last_accessed_at timestamptz NULL,
  expires_at       timestamptz NULL,
  is_active        boolean NOT NULL DEFAULT true,
  created_at       timestamptz NOT NULL DEFAULT now(),
  updated_at       timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT short_urls_code_key      UNIQUE (code),
  CONSTRAINT short_urls_code_format   CHECK (code ~ '^[A-Za-z0-9][A-Za-z0-9_-]{2,31}$'),
  CONSTRAINT short_urls_target_len    CHECK (char_length(target_url) <= 2048),
  CONSTRAINT short_urls_clicks_nonneg CHECK (clicks >= 0)
);
CREATE INDEX short_urls_user_created_idx ON short_urls (user_id, created_at DESC, id DESC);
CREATE INDEX short_urls_created_at_idx   ON short_urls (created_at DESC);

CREATE TABLE url_daily_clicks (
  url_id uuid   NOT NULL REFERENCES short_urls(id) ON DELETE CASCADE,
  day    date   NOT NULL,
  clicks bigint NOT NULL DEFAULT 0,
  PRIMARY KEY (url_id, day),
  CONSTRAINT url_daily_clicks_nonneg CHECK (clicks >= 0)
);

ALTER TABLE users            ENABLE ROW LEVEL SECURITY;
ALTER TABLE sessions         ENABLE ROW LEVEL SECURITY;
ALTER TABLE short_urls       ENABLE ROW LEVEL SECURITY;
ALTER TABLE url_daily_clicks ENABLE ROW LEVEL SECURITY;
