-------------------- ASSETS ---------------------

CREATE TYPE assetType AS ENUM (
'STOCK', 'CRYPTO', 'ETF'
);

CREATE TABLE assets
(
    id          BIGSERIAL PRIMARY KEY,
    ticker      TEXT        NOT NULL UNIQUE,
    name        TEXT        NOT NULL,
    type        assetType   NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL,
    modified_at TIMESTAMPTZ NOT NULL
);


-------------------- ASSET CANDLE RANGE ---------------------
CREATE TABLE asset_candle_range
(
    id         BIGSERIAL PRIMARY KEY,
    ticker     TEXT        NOT NULL UNIQUE,
    name       TEXT        NOT NULL,
    type       assetType   NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    start_time TIMESTAMPTZ NOT NULL,
    end_time   TIMESTAMPTZ NOT NULL
);
-------------------- CANDLES ---------------------
CREATE TABLE candles
(
    timestamp TIMESTAMPTZ    NOT NULL,
    asset_id  INT            NOT NULL REFERENCES assets (id) ON DELETE CASCADE,
    open      NUMERIC(18, 2) NOT NULL,
    high      NUMERIC(18, 2) NOT NULL,
    low       NUMERIC(18, 2) NOT NULL,
    close     NUMERIC(18, 2) NOT NULL,
    volume    NUMERIC(18, 8) NOT NULL,
    PRIMARY KEY (timestamp, asset_id)
);