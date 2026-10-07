# Short URL

A minimal, mobile-first URL shortener. Go backend, Vue 3 frontend, Postgres
(Supabase) storage.

## Stack

| Layer    | Choice                                            |
| -------- | ------------------------------------------------- |
| Backend  | Go 1.23, chi router, pgx/v5 (pool)                |
| Database | PostgreSQL 17 (Supabase), golang-migrate          |
| Auth     | Argon2id password hashing, opaque cookie sessions |
| Frontend | Vue 3 + TypeScript + Vite, lazy-loaded routes     |
| Tests    | Go unit + integration (real Postgres)             |

## Layout

```
backend/
  cmd/server/          entry point
  internal/
    config/            env parsing, defaults
    model/             domain types + DTOs
    validate/          URL/alias/expiry validation
    repository/        all SQL (pgx), keyset pagination
    service/           URL + auth + stats services, redirect cache
    middleware/        logging, recover, security headers, rate limit, CORS
    handler/           HTTP handlers
    router/            route table (order per PRD §5.2)
  migrations/          0001_init, 0002_supabase_lockdown
  test/                integration tests (build tag `integration`)
frontend/
  src/views|components|composables|services
docs/
  decisions.md         implementation decisions
  design-notes.md      design system record
```

## Running

Backend (API only):

```sh
cp backend/.env.example backend/.env   # fill DATABASE_URL
cd backend && go run ./cmd/server      # :8080
```

Frontend dev server (proxies `/api` to :8080):

```sh
cd frontend && npm install && npm run dev   # :5173
```

Production build served by the Go binary:

```sh
cd frontend && npm run build           # -> frontend/dist
cd backend && STATIC_DIR=../frontend/dist go run ./cmd/server
```

## Tests

```sh
cd backend
make lint              # gofmt + go vet
make test              # unit tests
# integration tests need a disposable local Postgres (PRD §17.1):
sudo -u postgres psql -c "CREATE USER surl_test PASSWORD 'surl_test';"
sudo -u postgres psql -c "CREATE DATABASE surl_test OWNER surl_test;"
migrate -path migrations -database "postgresql://surl_test:surl_test@localhost:5432/surl_test?sslmode=disable" up
make test-integration
```

## Migrations

```sh
cd backend
DATABASE_URL=... make migrate-up      # apply
DATABASE_URL=... make migrate-down    # roll back one
```

## Security notes

- All SQL lives in `internal/repository`, fully parameterized.
- Security headers (CSP, X-Content-Type-Options, Referrer-Policy, frame
  options) applied to every response; redirects get `Cache-Control: private,
max-age=60`.
- Sessions are opaque tokens (256-bit random), stored hashed, HttpOnly +
  SameSite=Lax (+ Secure in production).
- Redirect path: in-memory cache (TTL+LRU) → single query on miss → async
  click flush. Zero synchronous DB writes per redirect request.
- Rate limiting: per-IP token buckets, stricter buckets for auth endpoints
  plus a failure counter on login/register.

## Development

- **Format (frontend & docs):** `npm install` at the repo root, then
  `npm run format` / `npm run format:check` (Prettier, config in
  `.prettierrc.json`).
- **CI:** GitHub Actions (`.github/workflows/ci.yml`) runs gofmt, `go vet`,
  unit + integration tests (with a Postgres service), and the frontend
  typecheck + build on every push/PR.

## License

[ISC](LICENSE)
