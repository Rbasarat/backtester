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

func NewOrder(
	tradeID string,
	symbol string,
	price decimal.Decimal,
	quantity decimal.Decimal,
	orderType types.OrderType,
	side types.Side,
	createdAt time.Time,
) Order {
	return Order{
		tradeId:   tradeID,
		symbol:    symbol,
		price:     price,
		quantity:  quantity,
		orderType: orderType,
		side:      side,
		createdAt: createdAt,
	}
}
