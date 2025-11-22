package engine

import (
	"backtester/types"
	"time"

	"github.com/shopspring/decimal"
)

type DataFeedConfig struct {
	ticker   string
	interval types.Interval
	start    time.Time
	end      time.Time
	candles  []types.Candle
}

func NewDataFeedConfigs(feeds ...*DataFeedConfig) []*DataFeedConfig {
	return feeds
}

func NewDataFeedConfig(ticker string, interval types.Interval, start, end time.Time) *DataFeedConfig {
	return &DataFeedConfig{
		ticker:   ticker,
		interval: interval,
		start:    start,
		end:      end,
	}
}

type PortfolioConfig struct {
	initialCash       decimal.Decimal
	allowShortSelling bool
}

func NewPortfolioConfig(initialCash decimal.Decimal, allowShortSelling bool) *PortfolioConfig {
	return &PortfolioConfig{
		initialCash:       initialCash,
		allowShortSelling: allowShortSelling,
	}
}

type ExecutionConfig struct {
	interval   types.Interval
	barsBefore int
	barsAfter  int
	candles    map[string][]types.Candle
}

func NewExecutionConfig(executionInterval types.Interval, barsBefore, barsAfter int) *ExecutionConfig {
	return &ExecutionConfig{
		interval:   executionInterval,
		barsBefore: barsBefore,
		barsAfter:  barsAfter,
		candles:    make(map[string][]types.Candle),
	}
}

type ReportingConfig struct {
	sharpeRiskFreeRate decimal.Decimal
	printTrades        bool
	reportName         string
	filePath           string
}

func NewReportingConfig(sharpeRiskFreeRate decimal.Decimal, reportFile bool, reportName string, filePath string) *ReportingConfig {
	return &ReportingConfig{
		sharpeRiskFreeRate: sharpeRiskFreeRate,
		printTrades:        reportFile,
		reportName:         reportName,
		filePath:           filePath,
	}
}
