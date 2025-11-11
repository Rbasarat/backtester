package types

import (
	"time"

	"github.com/shopspring/decimal"
)

type PortfolioView struct {
	Cash      decimal.Decimal
	Positions map[string]PositionSnapshot
}

type PositionSnapshot struct {
	Symbol        string
	Side          Side
	Quantity      decimal.Decimal
	AvgEntryPrice decimal.Decimal
	LastPrice     decimal.Decimal
	EntryTime     time.Time
}
