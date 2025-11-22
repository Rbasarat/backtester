package types

import (
	"time"

	"github.com/shopspring/decimal"
)

type Order struct {
	Ticker       string
	Price        decimal.Decimal
	Quantity     decimal.Decimal
	OrderType    OrderType
	Side         Side
	SignalReason string
	CreatedAt    time.Time
}

func NewOrder(
	ticker string,
	price decimal.Decimal,
	quantity decimal.Decimal,
	orderType OrderType,
	side Side,
	signalReason string,
	createdAt time.Time,
) Order {
	return Order{
		Ticker:       ticker,
		Price:        price,
		Quantity:     quantity,
		OrderType:    orderType,
		Side:         side,
		SignalReason: signalReason,
		CreatedAt:    createdAt,
	}
}
