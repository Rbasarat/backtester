package engine

import (
	"backtester/types"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"time"
)

// writeTradesCSVFile writes trades to a CSV file at the given path.
func (e *Engine) writeTradesCSVFile(path string, trades []trade) error {
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
