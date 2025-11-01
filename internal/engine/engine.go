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

type Engine struct {
	db         dataStore
	backtester *backtester
}

func NewEngine(feeds []*DataFeed, strat strategy, db dataStore) *Engine {
	return &Engine{
		db:         db,
		backtester: newBacktester(feeds, strat),
	}
}

func (e *Engine) Run() error {
	// Load the data
	err := e.loadData()
	if err != nil {
		return err
	}
	// Do the run loop
	err = e.backtester.run()
	if err != nil {
		return err
	}

	return nil
}

func (e *Engine) loadData() error {
	ctx := context.Background()

	for _, feed := range e.backtester.feeds {
		asset, err := e.db.GetAssetByTicker(feed.Ticker, ctx)
		if err != nil {
			return err
		}
		cs, err := e.db.GetAggregates(asset.Id, feed.Interval, feed.Start, feed.End, ctx)
		if err != nil {
			return err
		}
		feed.candles = cs
	}
	return nil
}
