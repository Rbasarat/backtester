package engine

import (
	"context"
	"fmt"
)

type Engine struct {
	db                dataStore
	feeds             []*DataFeedConfig
	executionConfig   *ExecutionConfig
	portfolioConfig   *PortfolioConfig
	reportingConfig   *ReportingConfig
	strategy          strategy
	allocator         allocator
	broker            broker
	portfolio         *portfolio
	backtester        *backtester
	allowShortSelling bool
}

func NewEngine(
	feeds []*DataFeedConfig,
	executionConfig *ExecutionConfig,
	reportingConfig *ReportingConfig,
	strat strategy,
	sizer allocator,
	broker broker,
	portfolioConfig *PortfolioConfig, db dataStore) *Engine {

	// This is an ugly hack where backtester and portfolio depend on eachother.. we need to figure out how to expose currentTime
	initPortfolio := newPortfolio(portfolioConfig.initialCash, portfolioConfig.allowShortSelling)
	backtester := newBacktester(feeds, executionConfig, portfolioConfig, strat, sizer, broker, initPortfolio)
	initPortfolio.backtesterApi = backtester

	return &Engine{
		db:              db,
		feeds:           feeds,
		executionConfig: executionConfig,
		portfolioConfig: portfolioConfig,
		reportingConfig: reportingConfig,

		strategy:   strat,
		allocator:  sizer,
		broker:     broker,
		portfolio:  initPortfolio,
		backtester: backtester,
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

	// Inititalize strategy
	err = e.strategy.Init(e.backtester.portfolio)
	if err != nil {
		return err
	}
	err = e.allocator.Init(e.backtester.portfolio)
	if err != nil {
		return err
	}

	// Do the run loop
	err = e.backtester.run()
	if err != nil {
		return err
	}

	report := e.generateReport(e.backtester.start, e.backtester.curTime, e.backtester.portfolio)
	e.printReport(report)
	if e.reportingConfig.printTrades {
		filenameTrades := fmt.Sprintf("%s/%s_trades.csv", e.reportingConfig.filePath, e.reportingConfig.reportName)
		err = e.writeTradesCSVFile(filenameTrades, report.trades)
		if err != nil {
			return err
		}

		filenamePortfolio := fmt.Sprintf("%s/%s_portfolio.csv", e.reportingConfig.filePath, e.reportingConfig.reportName)
		err = e.writePortfolioCSVFile(filenamePortfolio, e.portfolio.snapshots)
		if err != nil {
			return err
		}

	}

	return nil
}

func (e *Engine) loadFeedData() error {
	ctx := context.Background()

	for _, feed := range e.backtester.feeds {
		asset, err := e.db.GetAssetByTicker(feed.ticker, ctx)
		if err != nil {
			return err
		}
		cs, err := e.db.GetAggregates(asset.Id, asset.Ticker, feed.interval, feed.start, feed.end, ctx)
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
		asset, err := e.db.GetAssetByTicker(feed.ticker, ctx)
		if err != nil {
			return err
		}
		cs, err := e.db.GetAggregates(asset.Id, asset.Ticker, feed.interval, feed.start, feed.end, ctx)
		if err != nil {
			return err
		}
		e.backtester.executionConfig.candles[feed.ticker] = cs
	}
	return nil
}
