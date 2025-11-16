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

func NewSignal(
	tradeID string,
	symbol string,
	side types.Side,
	price decimal.Decimal,
	reason string,
	createdAt time.Time,
) Signal {
	return Signal{
		tradeId:   tradeID,
		symbol:    symbol,
		side:      side,
		price:     price,
		reason:    reason,
		createdAt: createdAt,
	}
}
