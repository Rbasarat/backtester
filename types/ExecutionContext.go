package types

import (
	"time"
)

type ExecutionContext struct {
	Candles   map[string][]Candle
	Portfolio PortfolioView
	CurTime   time.Time
}
