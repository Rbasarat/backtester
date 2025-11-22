package types

import (
	"time"

	"github.com/shopspring/decimal"
)

type Order struct {
	Symbol       string
	Price        decimal.Decimal
	Quantity     decimal.Decimal
	OrderType    OrderType
	Side         Side
	SignalReason string
	CreatedAt    time.Time
}

func NewOrder(
	symbol string,
	price decimal.Decimal,
	quantity decimal.Decimal,
	orderType OrderType,
	side Side,
	signalReason string,
	createdAt time.Time,
) Order {
	return Order{
		Symbol:       symbol,
		Price:        price,
		Quantity:     quantity,
		OrderType:    orderType,
		Side:         side,
		SignalReason: signalReason,
		CreatedAt:    createdAt,
	}
}
