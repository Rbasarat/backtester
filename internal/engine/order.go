package engine

import (
	"backtester/types"
	"time"

	"github.com/shopspring/decimal"
)

type Order struct {
	tradeId   string
	symbol    string
	price     decimal.Decimal
	quantity  decimal.Decimal
	orderType types.OrderType
	side      types.Side
	createdAt time.Time
}

func NewOrder(tradeId, symbol string, price, quantity decimal.Decimal, orderType types.OrderType, side types.Side, createdAt time.Time) *Order {
	return &Order{
		tradeId, symbol, price, quantity, orderType, side, createdAt,
	}
}
