package types

import (
	"github.com/shopspring/decimal"
	"time"
)

type Portfolio struct {
	Cash         decimal.Decimal
	Positions    map[string]*Position
	OpenTrades   map[string][]*Trade
	ClosedTrades map[string][]*Trade
}

type Position struct {
	Symbol     string
	Side       Side
	Quantity   decimal.Decimal
	EntryPrice decimal.Decimal
	EntryTime  time.Time
	LastPrice  decimal.Decimal
}

func NewPortfolio(initialCash decimal.Decimal) *Portfolio {
	return &Portfolio{
		Cash:         initialCash,
		Positions:    make(map[string]*Position), //TODO: we can make this a list to allow for scaling in/out if needed later
		OpenTrades:   make(map[string][]*Trade),
		ClosedTrades: make(map[string][]*Trade),
	}
}

type PortfolioView struct {
	Cash      decimal.Decimal
	Positions map[string]PositionSnapshot
}

type PositionSnapshot struct {
	Symbol        string
	Side          Side
	Quantity      decimal.Decimal
	AvgEntryPrice decimal.Decimal
	LastPrice     decimal.Decimal
	EntryTime     time.Time
}

func (p *Portfolio) GetPortfolioSnapshot() PortfolioView {
	view := PortfolioView{
		Cash:      p.Cash,
		Positions: make(map[string]PositionSnapshot, len(p.Positions)),
	}

	for sym, pos := range p.Positions {
		view.Positions[sym] = PositionSnapshot{
			Symbol:    pos.Symbol,
			Side:      pos.Side,
			Quantity:  pos.Quantity,
			LastPrice: pos.LastPrice,
			EntryTime: pos.EntryTime,
		}
	}
	return view
}
