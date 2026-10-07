# Decisions

Small decisions made during implementation that were not fixed by the PRD.
One to three lines each; newest relevant at the bottom.

- D-add-01: `middleware.Trusted` type replaced by `[]*net.IPNet` — the PRD asked for CIDR support; standard library types keep it simple.
- D-add-02: Redirect cache implemented with `hashicorp/golang-lru/v2/expirable` — single dependency providing LRU + TTL, avoiding a hand-rolled eviction bug surface.
- D-add-03: `singleflight` is a ~30-line in-package implementation keyed by code, instead of `golang.org/x/sync/singleflight`, so concurrent misses share exactly one query and callers re-read the cache after waiting.
- D-add-04: Local Postgres 16 (apt) used as the integration-test database because Docker is not available on this host; PRD §17.1 allows "Postgres lokal". The test database is created by `make test-integration` setup documented in the README, never the production Supabase instance.
- D-add-05: Click buffer re-queues entries on flush failure (in-memory) so a transient DB error does not drop clicks; crash can still lose ≤ one interval, as accepted in PRD §8.4.
- D-add-06: The frontend adds `https://` to scheme-less input (D-17). The server additionally normalizes a scheme-less input the same way before strict validation, so direct API callers get consistent behavior.
- D-add-07: `argon2id` PHC parsing is hand-rolled (no dependency) but hash params remain configurable in one constant block; verification uses `subtle.ConstantTimeCompare`.
- D-add-08: Error pages (404/410) are rendered by Go with an inline `<style>`; the global CSP is tightened for app responses, and the error page CSP includes `'unsafe-inline'` for style only (D-09/§13.2 note).
