package types

import (
	"github.com/shopspring/decimal"
	"time"
)

type Candle struct {
	AssetId   int             `json:"id"`
	Open      decimal.Decimal `json:"open"`
	Close     decimal.Decimal `json:"close"`
	High      decimal.Decimal `json:"high" `
	Low       decimal.Decimal `json:"low"`
	Volume    decimal.Decimal `json:"volume"`
	Interval  Interval        `json:"interval"`
	Timestamp time.Time       `json:"timestamp"`
}
