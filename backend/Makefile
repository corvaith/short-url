.PHONY: dev test test-integration lint build migrate-up migrate-down

dev:
	go run ./cmd/server

test:
	go test -count=1 ./internal/...

# Integration tests run against a local Postgres test database.
# Requires APP_ENV=test and TEST_DATABASE_URL (see README).
test-integration:
	APP_ENV=test TEST_DATABASE_URL=$${TEST_DATABASE_URL:-postgresql://surl_test:surl_test@localhost:5432/surl_test?sslmode=disable} \
		go test -tags=integration -count=1 ./test/...

lint:
	go vet ./...
	@out=$$(gofmt -l .); if [ -n "$$out" ]; then echo "unformatted:"; echo "$$out"; exit 1; fi

build:
	go build -o bin/server ./cmd/server

MIGRATE_DB ?= $(DATABASE_URL)

migrate-up:
	migrate -path migrations -database "$(MIGRATE_DB)" up

migrate-down:
	migrate -path migrations -database "$(MIGRATE_DB)" down 1
