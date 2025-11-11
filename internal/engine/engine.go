package engine

import (
	"context"
)

type Engine struct {
	db              dataStore
	feeds           []*DataFeedConfig
	executionConfig ExecutionConfig
	strategy        strategy
	allocator       allocator
	broker          broker
	portfolio       *portfolio
	backtester      *backtester
}

func NewEngine(feeds []*DataFeedConfig, executionConfig ExecutionConfig, strat strategy, sizer allocator, broker broker, wallet *PortfolioConfig, db dataStore) *Engine {
	newPortfolio := newPortfolio(wallet.InitialCash)

	return &Engine{
		db:              db,
		feeds:           feeds,
		executionConfig: executionConfig,
		strategy:        strat,
		allocator:       sizer,
		broker:          broker,
		portfolio:       newPortfolio,
		backtester:      newBacktester(feeds, executionConfig, strat, sizer, broker, newPortfolio),
	}
}

func (e *Engine) Run() error {
	// Load the data
	// TODO: we can parallelize loading data if needed
	err := e.loadFeedData()
	if err != nil {
		return err
	}
	err = e.loadExecutionFeedData()
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

func (e *Engine) loadFeedData() error {
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
func (e *Engine) loadExecutionFeedData() error {
	ctx := context.Background()

	for _, feed := range e.feeds {
		asset, err := e.db.GetAssetByTicker(feed.Ticker, ctx)
		if err != nil {
			return err
		}
		cs, err := e.db.GetAggregates(asset.Id, feed.Interval, feed.Start, feed.End, ctx)
		if err != nil {
			return err
		}
		e.backtester.executionConfig.candles[feed.Ticker] = cs
	}
	return nil
}
