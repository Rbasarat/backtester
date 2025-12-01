package main

import (
	"backtester/internal/engine"
	"backtester/internal/repository"
	"backtester/strategies/donchian"
	"backtester/types"
	"log"
	"time"

	"github.com/shopspring/decimal"
)

const (
	dburl = "postgresql://moneymaker:moneymaker@localhost:5432/moneymaker"
)

func main() {
	db, err := repository.NewDatabase(dburl)
	if err != nil {
		log.Fatal(err)
	}

	start := time.Date(2015, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	feeds := getETFPortfolio(start, end, types.Day)

	eng := engine.NewEngine(
		feeds,
		engine.NewExecutionConfig(types.Hour, 24, 24),
		engine.NewReportingConfig(decimal.NewFromFloat(0.03), true, "donchian", "reports"),
		&donchian.Strategy{},
		donchian.NewLongOnlyAllocator(decimal.NewFromFloat(0.02)),
		&donchian.Broker{},
		engine.NewPortfolioConfig(decimal.NewFromFloat(50000), true),
		&db,
	)

	err = eng.Run()

	if err != nil {
		log.Fatal(err)
	}
}

func getETFPortfolio(start, end time.Time, interval types.Interval) []*engine.DataFeedConfig {
	return engine.NewDataFeedConfigs(
		engine.NewDataFeedConfig("QQQ", interval, start, end),
		engine.NewDataFeedConfig("SPY", interval, start, end),
	)
}

func getDiversifiedPortfolio(start, end time.Time, interval types.Interval) []*engine.DataFeedConfig {
	return engine.NewDataFeedConfigs(
		// ETF
		engine.NewDataFeedConfig("SPY", interval, start, end),
		engine.NewDataFeedConfig("QQQ", interval, start, end),
		// Consumer discretionary
		engine.NewDataFeedConfig("WMT", interval, start, end),
		engine.NewDataFeedConfig("NKE", interval, start, end),
		// Energy
		engine.NewDataFeedConfig("XOM", interval, start, end),
		engine.NewDataFeedConfig("CVX", interval, start, end),
		// Financials
		engine.NewDataFeedConfig("SOFI", interval, start, end),
		engine.NewDataFeedConfig("BAC", interval, start, end),
		// Health care
		engine.NewDataFeedConfig("JNJ", interval, start, end),
		engine.NewDataFeedConfig("UNH", interval, start, end),
		// Industrials
		engine.NewDataFeedConfig("RTX", interval, start, end),
		engine.NewDataFeedConfig("CAT", interval, start, end),
		// Tech
		engine.NewDataFeedConfig("NVDA", interval, start, end),
		engine.NewDataFeedConfig("INTC", interval, start, end),
		// Materials
		engine.NewDataFeedConfig("CDE", interval, start, end),
		engine.NewDataFeedConfig("HL", interval, start, end),
		// Metals/mining
		// Telecommunication
		engine.NewDataFeedConfig("T", interval, start, end),
		engine.NewDataFeedConfig("VZ", interval, start, end),
		// Utilities
		engine.NewDataFeedConfig("PCG", interval, start, end),
		engine.NewDataFeedConfig("OKLO", interval, start, end),
		// Real estate
		engine.NewDataFeedConfig("VICI", interval, start, end),
		engine.NewDataFeedConfig("AGNC", interval, start, end),
	)
}
