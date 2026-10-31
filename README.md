# Bybit Datasource Plugin

A Terminal datasrc WASM plugin for the [Bybit](https://bybit.com) V5 exchange.

## Features

- **Spot + USDT Perpetuals** market data
- **OHLCV** historical data and live WebSocket streaming
- **Account**, balances, positions, and order management
- **HMAC-SHA256** authentication (API key based)
- Full order type support: limit, market, stop, stop-limit, TP/SL, trailing stop
- TP/SL attachments directly on orders
- Native cancel-all support

## Build

```bash
./build.sh
```

Produces `plugin.wasm` (GOOS=wasip1 GOARCH=wasm).

## Configuration

| Field | Required | Description |
|-------|----------|-------------|
| `apiKey` | Yes | Bybit API key |
| `secretKey` | Yes | Bybit API secret |
| `accountType` | No | UNIFIED (default), CONTRACT, SPOT, or FUND |
| `testnet` | No | Use testnet endpoints |

## Supported Commands

| Command | Description |
|---------|-------------|
| `getMarkets` | Spot + linear instrument discovery |
| `getTimeframes` | 13 intervals (1m–1M) |
| `getOHLCV` | Historical kline data with start/end/limit |
| `getRecentOHLCV` | Latest N candles |
| `ohlcvStream` | Real-time WebSocket kline stream |
| `getAccount` | Wallet balance + account metadata |
| `getBalances` | Balance snapshot |
| `getPositions` | Open derivative positions (linear) |
| `getOrders` | Open orders (linear + spot) |
| `createOrders` | Place orders (limit, market, stop, TP/SL) |
| `cancelOrders` | Cancel by ID or cancel-all |
