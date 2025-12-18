package engine

import (
	"backtester/types"
	"time"

	"github.com/shopspring/decimal"
)

type InstrumentConfig struct {
	ticker   string
	interval types.Interval
	start    time.Time
	end      time.Time
	primary  TimeframeConfig
	context  []TimeframeConfig
}

type TimeframeConfig struct {
	interval types.Interval
	candles  []types.Candle
}

// TODO: we are doing a builder pattern for instruments.. lets do that for all config..
func Instruments(cfg ...*InstrumentConfig) []*InstrumentConfig {
	return cfg
}

func Instrument(ticker string, start, end time.Time, primaryInterval types.Interval) *InstrumentConfig {
	return &InstrumentConfig{
		ticker:   ticker,
		interval: primaryInterval,
		start:    start,
		end:      end,
		primary: TimeframeConfig{
			interval: primaryInterval,
		},
	}
}

func (c *InstrumentConfig) AddContext(interval types.Interval) *InstrumentConfig {
	c.context = append(c.context, TimeframeConfig{interval: interval})
	return c
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
