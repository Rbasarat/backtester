package engine

import (
	"backtester/types"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/shopspring/decimal"
)

// writeTradesCSVFile writes trades to a CSV file at the given path.
func (e *Engine) writePortfolioCSVFile(path string, views []types.PortfolioView) error {
	if err := os.MkdirAll(filepath.Dir(path), os.ModePerm); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create trades file: %w", err)
	}
	defer f.Close()

	return writePortfolioCSV(f, views)
}

func writePortfolioCSV(w io.Writer, views []types.PortfolioView) error {
	cw := csv.NewWriter(w)
	defer cw.Flush()

	// Header row
	header := []string{
		"snapshot_time",         // RFC3339
		"cash",                  // decimal
		"positions_value",       // decimal: sum(qty * last_market_price)
		"total_portfolio_value", // decimal: cash + positions_value
		"num_positions",         // int: count of positions
	}
	if err := cw.Write(header); err != nil {
		return fmt.Errorf("write header: %w", err)
	}

	for _, pv := range views {
		positionsValue := decimal.Zero
		numPositions := len(pv.Positions)

		for _, pos := range pv.Positions {
			// value = quantity * last market price
			value := pos.Quantity.Mul(pos.LastMarketPrice)
			positionsValue = positionsValue.Add(value)
		}

		totalValue := pv.Cash.Add(positionsValue)

		record := []string{
			pv.Time.Format(time.RFC3339),
			pv.Cash.StringFixed(2),
			positionsValue.StringFixed(2),
			totalValue.StringFixed(2),
			fmt.Sprintf("%d", numPositions),
		}

		if err := cw.Write(record); err != nil {
			return fmt.Errorf("write portfolio record: %w", err)
		}
	}

	// Check for any error from the csv.Writer
	if err := cw.Error(); err != nil {
		return fmt.Errorf("flush csv: %w", err)
	}

	return nil
}

// writeTradesCSVFile writes trades to a CSV file at the given path.
func (e *Engine) writeTradesCSVFile(path string, trades []trade) error {
	if err := os.MkdirAll(filepath.Dir(path), os.ModePerm); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create trades file: %w", err)
	}
	defer f.Close()

	return writeTradesCSV(f, trades)
}

// writeTradesCSV writes trades to any io.Writer as CSV.
// You can pass os.Stdout for debugging, or a file.
func writeTradesCSV(w io.Writer, trades []trade) error {
	cw := csv.NewWriter(w)
	defer cw.Flush()

	// Header row
	header := []string{
		"trade_id",
		"leg", // "buy" or "sell"
		"ticker",
		"side",
		"status",
		"total_filled_qty",
		"avg_fill_price",
		"total_fees",
		"remaining_qty",
		"num_fills",
		"reject_reason",
		"report_time", // RFC3339
	}
	if err := cw.Write(header); err != nil {
		return fmt.Errorf("write header: %w", err)
	}

	for i, t := range trades {
		tradeID := fmt.Sprintf("%d", i)

		if t.buy != nil {
			if err := writeExecutionRow(cw, tradeID, "buy", t.buy); err != nil {
				return err
			}
		}

		if t.sell != nil {
			if err := writeExecutionRow(cw, tradeID, "sell", t.sell); err != nil {
				return err
			}
		}
	}

	// Check for any error from the csv.Writer
	if err := cw.Error(); err != nil {
		return fmt.Errorf("flush csv: %w", err)
	}

	return nil
}

// Helper to convert a single ExecutionReport into one CSV row.
func writeExecutionRow(cw *csv.Writer, tradeID, leg string, er *types.ExecutionReport) error {
	record := []string{
		tradeID,
		leg,
		er.Ticker,
		string(er.Side),
		string(er.Status),
		er.TotalFilledQty.String(),
		er.AvgFillPrice.String(),
		er.TotalFees.String(),
		er.RemainingQty.String(),
		fmt.Sprintf("%d", len(er.Fills)),
		er.RejectReason,
		er.ReportTime.Format(time.RFC3339),
	}

	if err := cw.Write(record); err != nil {
		return fmt.Errorf("write record: %w", err)
	}
	return nil
}
