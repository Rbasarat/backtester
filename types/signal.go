package types

import (
	"time"

	"github.com/shopspring/decimal"
)

type Signal struct {
	Time   time.Time
	Symbol string
	Side   Side
	//Strength decimal.Decimal //TODO: in later version we will use this
	PriceAt decimal.Decimal
	Reason  string
}
