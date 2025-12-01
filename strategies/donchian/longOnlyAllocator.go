package donchian

import (
	"backtester/internal/engine"
	"backtester/types"

	"github.com/shopspring/decimal"
)

type LongOnlyAllocator struct {
	api             engine.PortfolioApi
	positionPercent decimal.Decimal
}

func NewLongOnlyAllocator(positionPercent decimal.Decimal) *LongOnlyAllocator {
	return &LongOnlyAllocator{
		positionPercent: positionPercent,
	}
}

func (a *LongOnlyAllocator) Init(api engine.PortfolioApi) error {
	a.api = api
	return nil
}
func (a *LongOnlyAllocator) Allocate(signals map[string][]types.Signal, view types.PortfolioView) []types.Order {
	if len(signals) == 0 {
		return nil
	}

	orders := make([]types.Order, 0)

	for ticker, signalPerTicker := range signals {
		curPos := view.Positions[ticker]

		// Skip tickers with 0 or more than 1 signal (double signal, etc.)
		if len(signalPerTicker) != 1 {
			continue
		}

		curSignal := signalPerTicker[0]

		// Case 1: flat
		if curPos.Quantity.IsZero() {
			// Long-only: only act on buy signals when flat
			if curSignal.Side != types.SideTypeBuy {
				continue
			}

			cashForSignal := view.Cash.Mul(a.positionPercent)
			qty := getQuantityForPrice(curSignal.Price, cashForSignal)
			if qty.IsZero() {
				continue
			}

			orders = append(orders, types.NewOrder(
				ticker, curSignal.Price, qty,
				types.TypeLimit, types.SideTypeBuy,
				"No existing position (long-only): "+curSignal.Reason,
				curSignal.CreatedAt,
			))
			continue
		}

		// Case 2: existing long
		if curPos.Quantity.IsPositive() {
			switch curSignal.Side {
			case types.SideTypeBuy:
				// same direction -> do nothing (no pyramiding here)
				continue
			case types.SideTypeSell:
				// Long-only: just close the long, do NOT open a short
				qty := curPos.Quantity

				orders = append(orders, types.NewOrder(
					ticker, curSignal.Price, qty,
					types.TypeLimit, types.SideTypeSell,
					"Closing long (long-only): "+curSignal.Reason,
					curSignal.CreatedAt,
				))
			}
			continue
		}

		// Case 3: existing short (should only happen from legacy runs)
		if curPos.Quantity.IsNegative() {
			switch curSignal.Side {
			case types.SideTypeSell:
				// Long-only: do nothing here (we're not adding to shorts)
				continue
			case types.SideTypeBuy:
				qty := curPos.Quantity.Abs()

				// Close short
				orders = append(orders, types.NewOrder(
					ticker, curSignal.Price, qty,
					types.TypeLimit, types.SideTypeBuy,
					"Closing short (long-only, switching to long): "+curSignal.Reason,
					curSignal.CreatedAt,
				))

				// Optionally open a new long (still long-only)
				cashForSignal := view.Cash.Mul(a.positionPercent)
				newQty := getQuantityForPrice(curSignal.Price, cashForSignal)
				if !newQty.IsZero() {
					orders = append(orders, types.NewOrder(
						ticker, curSignal.Price, newQty,
						types.TypeLimit, types.SideTypeBuy,
						"Opening new long after closing short: "+curSignal.Reason,
						curSignal.CreatedAt,
					))
				}
			}
			continue
		}
	}

	return orders
}

func getQuantityForPrice(stockPrice, capitalToUse decimal.Decimal) decimal.Decimal {
	if stockPrice.IsZero() {
		return decimal.Zero
	}
	return capitalToUse.Div(stockPrice).Floor()
}
