package types

import (
	"time"
)

type ExecutionContext struct {
	Candles   map[string]map[time.Time]Candle
	Portfolio PortfolioView
	CurTime   time.Time
}
