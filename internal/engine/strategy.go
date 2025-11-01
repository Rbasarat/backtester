package engine

import (
	"backtester/types"
)

type strategy interface {
	Init(api StrategyAPI) error
	OnCandle(candle types.Candle)
}

type StrategyAPI interface {
}
