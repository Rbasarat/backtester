package engine

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"
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
	logger            *slog.Logger
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
		logger:     slog.New(slog.NewTextHandler(os.Stdout, nil)),
	}
}

func (e *Engine) Run() error {
	start := time.Now()
	e.logger.Info("Starting backtest engine",
		slog.Time("start_time", start),
		slog.String("report_name", e.reportingConfig.reportName),
	)

	// Load feed data
	e.logger.Info("Loading feed data")
	if err := e.loadFeedData(); err != nil {
		e.logger.Error("Failed to load feed data", slog.Any("error", err))
		return err
	}
	e.logger.Info("Feed data loaded")

	// Load execution feed
	e.logger.Info("Loading execution feed data")
	if err := e.loadExecutionFeedData(); err != nil {
		e.logger.Error("Failed to load execution feed", slog.Any("error", err))
		return err
	}
	e.logger.Info("Execution feed data loaded")

	// Initialize strategy and allocator
	e.logger.Info("Initializing strategy and allocator")
	if err := e.strategy.Init(e.backtester.portfolio); err != nil {
		e.logger.Error("Strategy initialization failed", slog.Any("error", err))
		return err
	}
	if err := e.allocator.Init(e.backtester.portfolio); err != nil {
		e.logger.Error("Allocator initialization failed", slog.Any("error", err))
		return err
	}
	e.logger.Info("Strategy and allocator initialized successfully")

	// Run backtest
	e.logger.Info("Start backtesting")
	if err := e.backtester.run(); err != nil {
		e.logger.Error("Backtest run failed", slog.Any("error", err))
		return err
	}
	e.logger.Info("Backtest run completed",
		slog.Time("end_sim_time", e.backtester.curTime),
	)

	// Generate report
	e.logger.Info("Generating report")
	report := e.generateReport(e.backtester.start, e.backtester.curTime, e.backtester.portfolio)

	// Write trade and portfolio files
	if e.reportingConfig.printTrades {
		filenameTrades := fmt.Sprintf("%s/%s_trades.csv", e.reportingConfig.filePath, e.reportingConfig.reportName)
		e.logger.Info("Writing trades to CSV", slog.String("file", filenameTrades))
		if err := e.writeTradesCSVFile(filenameTrades, report.trades); err != nil {
			e.logger.Error("Failed to write trades CSV", slog.Any("error", err))
			return err
		}

		filenamePortfolio := fmt.Sprintf("%s/%s_portfolio.csv", e.reportingConfig.filePath, e.reportingConfig.reportName)
		e.logger.Info("Writing portfolio snapshots to CSV", slog.String("file", filenamePortfolio))
		if err := e.writePortfolioCSVFile(filenamePortfolio, e.portfolio.snapshots); err != nil {
			e.logger.Error("Failed to write portfolio CSV", slog.Any("error", err))
			return err
		}
	}

	e.logger.Info("Backtest completed successfully",
		slog.Duration("total_runtime", time.Since(start)),
		slog.String("report_name", e.reportingConfig.reportName),
	)

	e.printReport(report)

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
		cs, err := e.db.GetAggregates(asset.Id, asset.Ticker, e.executionConfig.interval, feed.start, feed.end, ctx)
		if err != nil {
			return err
		}
		e.backtester.executionConfig.candles[feed.ticker] = cs
	}
	return nil
}
