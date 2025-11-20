package types

import (
	"time"

	"github.com/shopspring/decimal"
)

type Order struct {
	TradeId   string
	Symbol    string
	Price     decimal.Decimal
	Quantity  decimal.Decimal
	OrderType OrderType
	Side      Side
	CreatedAt time.Time
}

func NewOrder(
	tradeID string,
	symbol string,
	price decimal.Decimal,
	quantity decimal.Decimal,
	orderType OrderType,
	side Side,
	createdAt time.Time,
) Order {
	return Order{
		TradeId:   tradeID,
		Symbol:    symbol,
		Price:     price,
		Quantity:  quantity,
		OrderType: orderType,
		Side:      side,
		CreatedAt: createdAt,
	}
}
