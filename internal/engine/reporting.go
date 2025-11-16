package engine

import (
	"time"

	"github.com/shopspring/decimal"
)

type Report struct {
	// Meta / period info
	StartDate   time.Time
	TotalPeriod time.Duration
	TotalTrades decimal.Decimal

	// Absolute & annualized performance
	NetProfit            decimal.Decimal
	AnnualizedNetProfit  decimal.Decimal
	NetAvgProfitPerTrade decimal.Decimal
	CAGR                 decimal.Decimal

	// Trade-level distribution metrics
	AvgWin  decimal.Decimal
	AvgLoss decimal.Decimal

	// Drawdown & loss streak metrics
	MaxDrawdown          decimal.Decimal
	MaxDrawdownPercent   decimal.Decimal
	MaxDrawdownDays      time.Duration
	MaxConsecutiveLosses time.Duration

	// Risk-adjusted metrics
	SharpeRatio  decimal.Decimal
	SortinaRatio decimal.Decimal
	ProfitFactor decimal.Decimal

	// Costs
	TotalFees decimal.Decimal

	// TODO: UPI (brent pentfold book)
}

func generateReport(results portfolio) *Report {
	return nil
}
