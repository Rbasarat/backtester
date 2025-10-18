package types

import "time"

type Interval string

const (
	OneMinute      Interval = "1"
	ThreeMinutes   Interval = "3"
	FiveMinutes    Interval = "5"
	FifteenMinutes Interval = "15"
	ThirtyMinutes  Interval = "30"
	Hour           Interval = "60"
	TwoHours       Interval = "120"
	FourHours      Interval = "240"
	Day            Interval = "D"
	Week           Interval = "W"
	Month          Interval = "M"
)

var IntervalToTime = map[Interval]time.Duration{
	OneMinute:      time.Minute,
	ThreeMinutes:   time.Minute * 3,
	FiveMinutes:    time.Minute * 5,
	FifteenMinutes: time.Minute * 15,
	ThirtyMinutes:  time.Minute * 30,
	Hour:           time.Hour,
	TwoHours:       time.Hour * 2,
	FourHours:      time.Hour * 4,
	Day:            time.Hour * 24,
}

var ConvertInterval = map[string]Interval{
	"1":   OneMinute,
	"3":   ThreeMinutes,
	"5":   FiveMinutes,
	"15":  FifteenMinutes,
	"30":  ThirtyMinutes,
	"60":  Hour,
	"120": TwoHours,
	"240": FourHours,
	"D":   Day,
	"W":   Week,
	"M":   Month,
}
