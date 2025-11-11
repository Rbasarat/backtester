package types

import "time"

type ExecutionContext struct {
	Candles map[string]map[time.Time]Candle
	CurTime time.Time
}
