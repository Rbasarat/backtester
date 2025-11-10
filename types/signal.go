package types

import (
	"github.com/shopspring/decimal"
	"time"
)

type Signal struct {
	Time   time.Time
	Symbol string
	Side   Side
	//Strength decimal.Decimal //TODO: in later version we will use this
	PriceAt decimal.Decimal
	Reason  string
}
