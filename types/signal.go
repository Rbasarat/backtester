package types

import (
	"time"

	"github.com/shopspring/decimal"
)

type Signal struct {
	Symbol string
	Side   Side
	//Strength decimal.Decimal //TODO: We can use this for Signal normalization later
	Price     decimal.Decimal
	Reason    string
	CreatedAt time.Time
}

func NewSignal(
	ticker string,
	side Side,
	price decimal.Decimal,
	reason string,
	createdAt time.Time,
) Signal {
	return Signal{
		Symbol:    ticker,
		Side:      side,
		Price:     price,
		Reason:    reason,
		CreatedAt: createdAt,
	}
}
