package engine

import (
	"backtester/types"
	"errors"
	"sort"

	"github.com/shopspring/decimal"
)

var UnknownSideErr = errors.New("unknown fill side")
var InsufficientBalanceErr = errors.New("insufficient balance when applying order fill")
var ShortSellNotAllowedErr = errors.New("short sell not allowed, broker sold more stock than in portfolio")
var NegativeQtyErr = errors.New("negative qty for fill is not allowed, please set the side to SideTypeSell")

type portfolio struct {
	cash              decimal.Decimal
	positions         map[string]*Position
	executions        []types.ExecutionReport
	realizedPnL       decimal.Decimal
	snapshots         []types.PortfolioView
	backtesterApi     backtesterApi
	allowShortSelling bool
}

func (p *portfolio) GetFillsForTicker(ticker string) []types.ExecutionReport {
	var reports []types.ExecutionReport
	for _, report := range p.executions {
		if report.Ticker == ticker {
			reports = append(reports, report)
		}
	}
	return reports
}

type Position struct {
	Ticker             string
	Quantity           decimal.Decimal
	AvgCost            decimal.Decimal
	LastExecutionPrice decimal.Decimal
}

func newPortfolio(initialCash decimal.Decimal, allowShortSelling bool) *portfolio {
	return &portfolio{
		cash:              initialCash,
		positions:         make(map[string]*Position),
		allowShortSelling: allowShortSelling,
	}
}

func (p *portfolio) GetPortfolioSnapshot() types.PortfolioView {
	view := types.PortfolioView{
		Cash:      p.cash,
		Positions: make(map[string]types.PositionSnapshot),
		Time:      p.backtesterApi.getCurrentTime(),
	}

	for sym, pos := range p.positions {

		view.Positions[sym] = types.PositionSnapshot{
			Ticker:          pos.Ticker,
			Quantity:        pos.Quantity,
			AvgEntryPrice:   pos.AvgCost,
			LastMarketPrice: p.backtesterApi.getLastPriceForTicker(sym),
		}
	}
	return view
}

func (p *portfolio) processExecutions(execs []types.ExecutionReport) error {
	if len(execs) == 0 {
		return nil
	}

	sort.Slice(execs, func(i, j int) bool {
		return execs[i].ReportTime.Before(execs[j].ReportTime)
	})

	for _, er := range execs {
		if len(er.Fills) == 0 {
			continue
		}

		// Validate side once per execution report
		if er.Side != types.SideTypeBuy && er.Side != types.SideTypeSell {
			return UnknownSideErr
		}

		// Precompute side sign: +1 for BUY, -1 for SELL
		sideSign := decimal.NewFromInt(1)
		if er.Side == types.SideTypeSell {
			sideSign = sideSign.Neg()
		}

		fills := append([]types.Fill(nil), er.Fills...)
		sort.Slice(fills, func(i, j int) bool {
			return fills[i].Time.Before(fills[j].Time)
		})

		pos := p.positions[er.Ticker]
		if pos == nil {
			// Create new position if it doesn't exist
			pos = &Position{Ticker: er.Ticker}
			p.positions[er.Ticker] = pos
		}

		for _, fill := range fills {
			if fill.Quantity.IsNegative() {
				return NegativeQtyErr
			}

			// Set qty to negative if we have types.SideTypeSell
			quantity := fill.Quantity.Mul(sideSign)

			cashDelta := fill.Price.Mul(quantity).Neg()
			newCash := p.cash.Add(cashDelta).Sub(fill.Fee)

			if newCash.LessThan(decimal.Zero) {
				return InsufficientBalanceErr
			}
			p.cash = newCash

			// Position quantity update
			oldQty := pos.Quantity
			newQty := oldQty.Add(quantity)

			if !p.allowShortSelling && newQty.IsNegative() {
				return ShortSellNotAllowedErr
			}

			// Average cost logic based on old/new side and size
			switch {
			case sameSide(oldQty, newQty):
				// Increasing an existing position on the same side
				absOld := oldQty.Abs()
				absNew := newQty.Abs()
				absAdd := quantity.Abs()

				if absNew.GreaterThan(absOld) && !absAdd.IsZero() {
					pos.AvgCost = weightedAvg(pos.AvgCost, absOld, fill.Price, absAdd)
				}
				pos.Quantity = newQty

			case oldQty.IsZero():
				// Opening a new position from flat
				pos.Quantity = newQty
				pos.AvgCost = fill.Price

			case newQty.IsZero():
				// Fully closed position
				pos.Quantity = decimal.Zero
				pos.AvgCost = decimal.Zero

			default:
				// Flipped side (e.g. went from long to short or vice versa)
				pos.Quantity = newQty
				pos.AvgCost = fill.Price
			}

			pos.LastExecutionPrice = fill.Price
		}

		// Store the full execution report for audit / reporting
		p.executions = append(p.executions, er)
	}

	return nil
}

func sameSide(a, b decimal.Decimal) bool {
	return (a.GreaterThan(decimal.Zero) && b.GreaterThan(decimal.Zero)) ||
		(a.LessThan(decimal.Zero) && b.LessThan(decimal.Zero))
}

func weightedAvg(existingAvgPrice, existingQty, newPrice, newQty decimal.Decimal) decimal.Decimal {
	if existingQty.IsZero() {
		return newPrice
	}
	return existingAvgPrice.Mul(existingQty).
		Add(newPrice.Mul(newQty)).
		Div(existingQty.Add(newQty))
}
