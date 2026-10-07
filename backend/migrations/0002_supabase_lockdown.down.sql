-- 0002_supabase_lockdown.down.sql
-- No-op: re-granting default Data API privileges is unnecessary; leaving
-- tables locked down is the safe direction on rollback.
SELECT 1;
