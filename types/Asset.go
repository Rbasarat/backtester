package types

import (
	"time"
)

type AssetType string

const (
	AssetTypeStock  AssetType = "STOCK"
	AssetTypeCrypto AssetType = "CRYPTO"
	AssetTypeEtf    AssetType = "ETF"
)

type Asset struct {
	Id         int       `json:"id"`
	Ticker     string    `json:"ticker"`
	Name       string    `json:"name"`
	Type       AssetType `json:"type"`
	CreatedAt  time.Time `json:"createdAt"`
	ModifiedAt time.Time `json:"modifiedAt"`
}

type AssetCandles struct {
	StartDate time.Time `json:"startDate"`
	EndDate   time.Time `json:"endDate"`
}
