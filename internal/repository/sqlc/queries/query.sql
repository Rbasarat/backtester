-------------------- ASSETS ---------------------
-- Get assets
-- name: GetAssets :many
SELECT *
FROM assets;

-- Get assets by type
-- name: GetAssetsByType :many
SELECT *
FROM assets
WHERE type = $1;

-- Get asset by ticker
-- name: GetAssetByTicker :one
SELECT *
FROM assets
WHERE ticker = $1;

-- Get asset by id
-- name: GetAssetById :one
SELECT *
FROM assets
WHERE id = $1;

-------------------- CANDLES ---------------------
-- Get min/max candle by assetId
-- name: GetCandleRangeByTicker :one
SELECT *
FROM asset_candle_range
WHERE ticker = $1;

-------------------- AGGREGATES ---------------------
-- Get aggregate
-- name: GetAggregates :many
SELECT time_bucket($1, c.timestamp)::timestamptz AS bucket,
    asset_id,
    first(open, c.timestamp)::numeric                   AS open,
    max(high)::numeric                                  AS high,
    min(low)::numeric                                   AS low,
    last(close, c.timestamp)::numeric                   AS close,
    sum(volume)::numeric                                AS volume
FROM candles c
where c.asset_id = $2
  and c.timestamp:: date BETWEEN @startTime
  AND @endTime
GROUP BY bucket, asset_id
ORDER BY bucket ASC;