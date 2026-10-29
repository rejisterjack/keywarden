package testlock

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// Acquire holds a session advisory lock so integration packages do not share one database at once.
func Acquire(url string) (func(), error) {
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, url)
	if err != nil {
		return nil, err
	}
	if _, err := conn.Exec(ctx, `SELECT pg_advisory_lock(84211001)`); err != nil {
		conn.Close(ctx)
		return nil, fmt.Errorf("test lock: %w", err)
	}
	return func() {
		_, _ = conn.Exec(context.Background(), `SELECT pg_advisory_unlock(84211001)`)
		conn.Close(context.Background())
	}, nil
}
