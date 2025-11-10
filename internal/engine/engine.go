package engine

import (
	"backtester/types"
	"context"
)

type Engine struct {
	db         dataStore
	feeds      []*DataFeedConfig
	strategy   strategy
	allocator  allocator
	broker     broker
	portfolio  *types.Portfolio
	backtester *backtester
}

func NewEngine(feeds []*DataFeedConfig, strat strategy, sizer allocator, broker broker, wallet *PortfolioConfig, db dataStore) *Engine {
	newPortfolio := types.NewPortfolio(wallet.InitialCash)
	return &Engine{
		db:         db,
		feeds:      feeds,
		strategy:   strat,
		allocator:  sizer,
		broker:     broker,
		portfolio:  newPortfolio,
		backtester: newBacktester(feeds, strat, sizer, broker, newPortfolio),
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
