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

	feeds := getDonchianPortfolio(start, end, types.Week)

	eng := engine.NewEngine(
		feeds,
		engine.NewExecutionConfig(types.Hour, 24, 24),
		engine.NewReportingConfig(decimal.NewFromFloat(0.03), true, "donchian", "reports"),
		&donchian.Strategy{},
		donchian.NewLongOnlyAllocator(decimal.NewFromFloat(0.1)),
		&donchian.Broker{},
		engine.NewPortfolioConfig(decimal.NewFromFloat(2000), true),
		&db,
	)

	err = eng.Run()

	if err != nil {
		log.Fatal(err)
	}
}

func getETFPortfolio(start, end time.Time, interval types.Interval) []*engine.InstrumentConfig {
	return engine.Instruments(
		engine.Instrument("AMD", start, end, interval).AddContext(types.Week),
		engine.Instrument("COST", start, end, interval),
	)
}

func getForexPortfolio(start, end time.Time, interval types.Interval) []*engine.InstrumentConfig {
	return engine.Instruments(
		engine.Instrument("EURUSD", start, end, interval),
		engine.Instrument("USDJPY", start, end, interval),
		engine.Instrument("GBPUSD", start, end, interval),
		engine.Instrument("AUDUSD", start, end, interval),
		engine.Instrument("USDCAD", start, end, interval),
	)
}

func getDonchianPortfolio(start, end time.Time, interval types.Interval) []*engine.InstrumentConfig {
	return engine.Instruments(
		engine.Instrument("AMD", start, end, types.Hour).AddContext(types.Week),
	)
}

func getDiversifiedPortfolio(start, end time.Time, interval types.Interval) []*engine.InstrumentConfig {
	return engine.Instruments(
		// ETF
		engine.Instrument("SPY", start, end, interval),
		engine.Instrument("QQQ", start, end, interval),
		// Consumer discretionary
		engine.Instrument("WMT", start, end, interval),
		engine.Instrument("NKE", start, end, interval),
		// Energy
		engine.Instrument("XOM", start, end, interval),
		engine.Instrument("CVX", start, end, interval),
		// Financials
		engine.Instrument("SOFI", start, end, interval),
		engine.Instrument("BAC", start, end, interval),
		// Health care
		engine.Instrument("JNJ", start, end, interval),
		engine.Instrument("UNH", start, end, interval),
		// Industrials
		engine.Instrument("RTX", start, end, interval),
		engine.Instrument("CAT", start, end, interval),
		// Tech
		engine.Instrument("NVDA", start, end, interval),
		engine.Instrument("INTC", start, end, interval),
		// Materials
		engine.Instrument("CDE", start, end, interval),
		engine.Instrument("HL", start, end, interval),
		// Metals/mining
		// Telecommunication
		engine.Instrument("T", start, end, interval),
		engine.Instrument("VZ", start, end, interval),
		// Utilities
		engine.Instrument("PCG", start, end, interval),
		engine.Instrument("OKLO", start, end, interval),
		// Real estate
		engine.Instrument("VICI", start, end, interval),
		engine.Instrument("AGNC", start, end, interval),
	)
}
