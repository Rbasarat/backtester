package types

import (
	"time"

	"github.com/shopspring/decimal"
)

type PortfolioView struct {
	Cash      decimal.Decimal
	Positions map[string]PositionSnapshot
	Time      time.Time
}

type PositionSnapshot struct {
	Ticker          string
	Quantity        decimal.Decimal
	AvgEntryPrice   decimal.Decimal
	LastMarketPrice decimal.Decimal
}
