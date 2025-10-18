package repository

import (
	"backtester/types"
	"context"
	"database/sql"
	"errors"
)

// GetAssetByTicker retrieves an types.Asset by its ticker symbol.
func (db *Database) GetAssetByTicker(ctx context.Context, ticker string) (*types.Asset, error) {
	asset, err := db.assets.GetAssetByTicker(ctx, ticker)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrAssetNotFound
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
