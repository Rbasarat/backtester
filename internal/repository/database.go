package repository

import (
	sqlc "backtester/internal/repository/sqlc/generated"
	"context"
	"errors"
	"fmt"

	pgxdecimal "github.com/jackc/pgx-shopspring-decimal"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Global error declarations.
var (
	ErrIntervalNotSupported = errors.New("timeframe not supported")
	ErrAssetNotFound        = errors.New("not found in datasource")
	ErrNoCandles            = errors.New("no candles found in datasource")
)

type assetsRepository interface {
	GetAssetByTicker(ctx context.Context, ticker string) (sqlc.Asset, error)
}
type candlesRepository interface {
	GetAggregates(ctx context.Context, arg sqlc.GetAggregatesParams) ([]sqlc.GetAggregatesRow, error)
}

// Database struct that holds the database connection and queries.
type Database struct {
	assets  assetsRepository
	candles candlesRepository
	conn    *pgxpool.Pool
}

// NewDatabase creates a new Database instance and verifies connectivity.
func NewDatabase(dbURL string) (Database, error) {
	config, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		return Database{}, fmt.Errorf("parse config: %w", err)
	}
	// Register shopspring decimal
	config.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		pgxdecimal.Register(conn.TypeMap())
		return nil
	}

	conn, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return Database{}, err
	}
	// Ensure the connection is established.
	if err := conn.Ping(context.Background()); err != nil {
		return Database{}, err
	}

	queries := sqlc.New(conn)
	return Database{
		assets:  queries,
		candles: queries,
		conn:    conn}, nil
}
