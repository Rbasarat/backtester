package repository

import (
	sqlc "backtester/internal/repository/sqlc/generated"
	"backtester/types"
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"
)

type mockAssetsRepository struct {
	sqlError error
}

func TestDatabase_GetAssetByTicker(t *testing.T) {
	type args struct {
		ticker string
	}
	tests := []struct {
		name    string
		args    args
		want    *types.Asset
		sqlcErr error
		wantErr error
	}{
		{"should throw ErrAssetNotFound", args{"AAPL"}, nil, sql.ErrNoRows, ErrAssetNotFound},
		{"should return asset", args{"AAPL"}, &types.Asset{Ticker: "AAPL", Id: 1}, nil, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := &Database{
				assets: mockAssetsRepository{
					sqlError: tt.sqlcErr,
				},
			}
			got, err := db.GetAssetByTicker(context.Background(), tt.args.ticker)
			if err != nil {
				if !errors.Is(err, ErrAssetNotFound) {
					t.Errorf("GetAssetByTicker() error = %v, wantErr %v", err, tt.wantErr)
				}
				return
			}
			if got.Ticker != tt.want.Ticker {
				t.Errorf("GetAssetByTicker() ticker = %v, want %v", got, tt.want)
			}
			if got.Id != tt.want.Id {
				t.Errorf("GetAssetByTicker() id = %v, want %v", got, tt.want)
			}
		})
	}
}

func (m mockAssetsRepository) GetAssetByTicker(_ context.Context, ticker string) (sqlc.Asset, error) {
	if m.sqlError != nil {
		return sqlc.Asset{}, m.sqlError
	}
	curTime := time.UnixMilli(1)
	return sqlc.Asset{
		ID:         1,
		Ticker:     ticker,
		Name:       "Apple",
		Type:       sqlc.AssettypeSTOCK,
		CreatedAt:  &curTime,
		ModifiedAt: &curTime,
	}, nil
}
