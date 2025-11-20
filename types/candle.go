package types

import (
	"time"

	"github.com/shopspring/decimal"
)

type Candle struct {
	AssetId   int             `json:"id"`
	Ticker    string          `json:"ticker"`
	Open      decimal.Decimal `json:"open"`
	Close     decimal.Decimal `json:"close"`
	High      decimal.Decimal `json:"high" `
	Low       decimal.Decimal `json:"low"`
	Volume    decimal.Decimal `json:"volume"`
	Interval  Interval        `json:"interval"`
	Timestamp time.Time       `json:"timestamp"`
}
