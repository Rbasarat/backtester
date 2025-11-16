package engine

import (
	"backtester/types"
	"time"

	"github.com/shopspring/decimal"
)

type Signal struct {
	tradeId string
	symbol  string
	side    types.Side
	//Strength decimal.Decimal //TODO: We can use this for Signal normalization later
	price     decimal.Decimal
	reason    string
	createdAt time.Time
}

func NewSignal(tradeId, symbol, reason string, side types.Side, price decimal.Decimal, createdAt time.Time) *Signal {
	return &Signal{
		tradeId, symbol, side, price, reason, createdAt,
	}
}
