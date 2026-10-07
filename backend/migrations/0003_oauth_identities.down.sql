DROP TABLE IF EXISTS oauth_identities;
-- restore NOT NULL only if no NULL hashes exist
UPDATE users SET password_hash = '' WHERE password_hash IS NULL;
ALTER TABLE users ALTER COLUMN password_hash SET NOT NULL;
