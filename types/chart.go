package types

import (
	"time"
)

type Chart struct {
	ID       int       `json:"id"`
	Ticker   string    `json:"ticker"`
	Candles  []Candle  `json:"candles"`
	Start    time.Time `json:"start"`
	End      time.Time `json:"end"`
	Interval Interval  `json:"interval"`
}
