package engine

import (
	"backtester/types"
	"context"
	"time"
)

type dataStore interface {
	GetAssetByTicker(ticker string, ctx context.Context) (*types.Asset, error)
	GetAggregates(assetId int, interval types.Interval, start, end time.Time, ctx context.Context) ([]types.Candle, error)
}

type strategy interface {
	Init(api PortfolioApi) error
	OnCandle(candle types.Candle) []types.Signal
}
type allocator interface {
	Allocate(signals []types.Signal, view types.PortfolioView) []types.Order
}
type PortfolioApi interface {
	GetPortfolioSnapshot() types.PortfolioView
}
type broker interface {
	Execute([]types.Order)
}
