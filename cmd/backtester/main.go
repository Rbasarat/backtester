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

	feeds := getEtfPortfolio(start, end, types.Day, types.Week)
	//feeds := getDonchianStockPortfolio(donchianUniverseByYear[2016], start, end, types.Day, types.Week)

	eng := engine.NewEngine(
		feeds,
		engine.NewExecutionConfig(types.Hour, 24, 24),
		engine.NewReportingConfig(decimal.NewFromFloat(0.03), true, "", "/Users/rbasarat/PyCharmMiscProject/backtest/results"),
		&donchian.Strategy{},
		donchian.NewLongOnlyAllocator(decimal.NewFromFloat(0.1)),
		&donchian.Broker{},
		engine.NewPortfolioConfig(decimal.NewFromFloat(5000), true),
		&db,
	)

	err = eng.Run()
	if err != nil {
		log.Fatal(err)
	}
}

func getDonchianStockPortfolio(tickers []string, start, end time.Time, fast, slow types.Interval) []*engine.InstrumentConfig {
	var instruments []*engine.InstrumentConfig
	for _, ticker := range tickers {
		instruments = append(instruments, engine.Instrument(ticker, start, end, fast).AddContext(slow))
	}
	return instruments
}

func getEtfPortfolio(start, end time.Time, intervalFast, intervalSlow types.Interval) []*engine.InstrumentConfig {
	return engine.Instruments(
		engine.Instrument("IVV", start, end, intervalFast).AddContext(intervalSlow),
		engine.Instrument("SPY", start, end, intervalFast).AddContext(intervalSlow),
		engine.Instrument("VOO", start, end, intervalFast).AddContext(intervalSlow),
		engine.Instrument("QQQ", start, end, intervalFast).AddContext(intervalSlow),
		engine.Instrument("VTI", start, end, intervalFast).AddContext(intervalSlow),
		engine.Instrument("IWM", start, end, intervalFast).AddContext(intervalSlow),
		engine.Instrument("SLV", start, end, intervalFast).AddContext(intervalSlow),
		engine.Instrument("GLD", start, end, intervalFast).AddContext(intervalSlow),
		//engine.Instrument("SENS", start, end, intervalFast).AddContext(intervalSlow),
		//engine.Instrument("AAL", start, end, intervalFast).AddContext(intervalSlow),
		//engine.Instrument("PLTR", start, end, intervalFast).AddContext(intervalSlow),
		//engine.Instrument("FCEL", start, end, intervalFast).AddContext(intervalSlow),
		//engine.Instrument("GME", start, end, intervalFast).AddContext(intervalSlow),
		//engine.Instrument("PLUG", start, end, intervalFast).AddContext(intervalSlow),
		//engine.Instrument("FCEL", start, end, intervalFast).AddContext(intervalSlow),
	)
}

var donchianUniverseByYear = map[int][]string{
	2016: {"BAC", "SUNE", "KMI", "GE", "FCX", "PFE", "AAPL", "F"},
	2017: {"KMI", "SIRI", "FCX", "F", "GE", "PFE", "BAC", "AAPL"},
	2018: {"BAC", "AMD", "T", "GE", "MU", "F", "SNAP", "SIRI"},
	2019: {"AMD", "BAC", "GE", "F", "SNAP", "CMCSA", "WFC", "SIRI"},
	2020: {"AMD", "BAC", "GE", "F", "T", "SNAP", "ACB", "SIRI"},
	2021: {"GE", "F", "AAPL", "AAL", "PLTR", "SNDL", "WFC", "TOPS"},
	2022: {"SNDL", "AMC", "AAPL", "F", "CTRM", "F", "PLTR", "BAC"},
	2023: {"BAC", "T", "KMI", "GE", "FCX", "PFE", "AAPL", "F"},
	2024: {"TSLA", "SNAP", "F", "PLTR", "BAC", "T", "AAL", "CGC"},
	2025: {"NVDA", "MARA", "F", "PLUG", "RIVN", "BAC", "PFE", "T"},
}
