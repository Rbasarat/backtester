package repository

import (
	"backtester/types"
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// GetAssetByTicker retrieves a types.Asset by its ticker.
func (db *Database) GetAssetByTicker(ticker string, ctx context.Context) (*types.Asset, error) {
	asset, err := db.assets.GetAssetByTicker(ctx, ticker)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("ticker %s %w", ticker, ErrAssetNotFound)
		}
		return nil, err
	}
	return &types.Asset{
		Id:         int(asset.ID),
		Ticker:     asset.Ticker,
		Name:       asset.Name,
		Type:       types.AssetType(asset.Type),
		CreatedAt:  *asset.CreatedAt,
		ModifiedAt: *asset.ModifiedAt,
	}, nil
}
