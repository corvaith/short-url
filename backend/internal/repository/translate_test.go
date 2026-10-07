package repository

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestTranslate(t *testing.T) {
	pe := &pgconn.PgError{Code: "23505"}
	if !errors.Is(translate(pe), ErrConflict) {
		t.Fatal("translate did not map 23505 to ErrConflict")
	}
}
