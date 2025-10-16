# Backtester v2

A modular backtesting framework for algorithmic trading with **hot*swappable strategies** and a **simple, reproducible
core**.

## Overview

* **Backtest Engine:** runs simulations, manages data, portfolio, and metrics.
* **Strategy Module:** defines buy/sell signals, position sizing, and risk rules.
* Strategies can declare their own subscribed symbols and bar sizes.

## Design Highlights

* Bar*close fills (no latency)
* Zero costs/slippage (for now)
* Single*currency PnL
* Deterministic runs (fixed seeds)
* Terminal*based output

## Metrics (TODO: tweak this)
* Trade duration
* Number of trades
* Final Cash
* Peak Cash
* Min Cash
* Total Fees Paid
* Total Return
* Win Rate
* Best Trade Profit
* Worst Trade Profit
* Shortest Trade Duration
* Longest Trade Duration