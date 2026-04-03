package database

import (
	"context"
	"database/sql"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type Options struct {
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

type QueryTimeouts struct {
	Read   time.Duration
	Write  time.Duration
	Ingest time.Duration
}

func Open(databaseURL string, options Options) (*sql.DB, error) {
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return nil, err
	}

	if options.MaxOpenConns > 0 {
		db.SetMaxOpenConns(options.MaxOpenConns)
	}
	if options.MaxIdleConns > 0 {
		db.SetMaxIdleConns(options.MaxIdleConns)
	}
	if options.ConnMaxLifetime > 0 {
		db.SetConnMaxLifetime(options.ConnMaxLifetime)
	}
	if options.ConnMaxIdleTime > 0 {
		db.SetConnMaxIdleTime(options.ConnMaxIdleTime)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}

	return db, nil
}

func WithTimeout[T any](ctx context.Context, timeout time.Duration, fn func() (T, error)) (T, error) {
	var zero T

	if ctx == nil {
		ctx = context.Background()
	}

	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	type result struct {
		value T
		err   error
	}

	ch := make(chan result, 1)
	go func() {
		value, err := fn()
		ch <- result{value: value, err: err}
	}()

	select {
	case <-ctx.Done():
		return zero, ctx.Err()
	case result := <-ch:
		return result.value, result.err
	}
}

func WithTimeoutVoid(ctx context.Context, timeout time.Duration, fn func() error) error {
	_, err := WithTimeout(ctx, timeout, func() (struct{}, error) {
		return struct{}{}, fn()
	})
	return err
}
