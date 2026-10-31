package bybit

import (
	"fmt"
	"net/url"
	"strconv"

	"github.com/plusev-terminal/go-plugin-common/v2/datasrc/exchange"
	rt "github.com/plusev-terminal/go-plugin-common/v2/requester/types"
	tt "github.com/plusev-terminal/go-plugin-common/v2/trading"
	utils "github.com/plusev-terminal/go-plugin-common/v2/wasmutils"
)


// GetOHLCV fetches historical OHLCV data from Bybit using /v5/market/kline.
// This is a public endpoint — no authentication required.
func (c *Client) GetOHLCV(params exchange.GetOHLCVParams) ([]tt.OHLCVRecord, error) {
	symbol := params.Market.Symbol
	if symbol == "" {
		return nil, fmt.Errorf("market.symbol is required")
	}

	category := AssetTypeToCategory(params.Market.AssetType)

	interval, err := c.convertTimeframeToBybitInterval(params.Timeframe)
	if err != nil {
		c.log.ErrorWithData("Invalid timeframe format", map[string]any{"timeframe": params.Timeframe, "error": err})
		return nil, err
	}

	tf, err := tt.TimeframeFromString(params.Timeframe)
	if err != nil {
		return nil, fmt.Errorf("invalid timeframe format: %w", err)
	}

	// Set limit with default
	limit := params.Limit
	if limit <= 0 {
		limit = DefaultLimit
	}
	if limit > BybitMaxLimit {
		limit = BybitMaxLimit
	}

	// Build query params
	u, err := url.Parse(c.baseURL + "/v5/market/kline")
	if err != nil {
		return nil, fmt.Errorf("failed to parse kline url: %w", err)
	}

	q := u.Query()
	q.Set("category", category)
	q.Set("symbol", symbol)
	q.Set("interval", interval)

	now, _ := utils.Now()

	switch {
	case params.StartTime == nil && params.EndTime == nil:
		// Serve "limit" candles back from now
		q.Set("end", strconv.FormatInt(now.UnixMilli(), 10))
		candleMs := int64(tf.ToMinutes()) * 60 * 1000
		q.Set("start", strconv.FormatInt(now.UnixMilli()-int64(limit)*candleMs, 10))

	case params.StartTime != nil && params.EndTime == nil:
		// Serve "limit" candles forward from start
		q.Set("start", strconv.FormatInt(params.StartTime.UnixMilli(), 10))
		candleMs := int64(tf.ToMinutes()) * 60 * 1000
		q.Set("end", strconv.FormatInt(params.StartTime.UnixMilli()+int64(limit)*candleMs, 10))

	case params.StartTime == nil && params.EndTime != nil:
		// Serve "limit" candles back ending at end
		q.Set("end", strconv.FormatInt(params.EndTime.UnixMilli(), 10))
		candleMs := int64(tf.ToMinutes()) * 60 * 1000
		q.Set("start", strconv.FormatInt(params.EndTime.UnixMilli()-int64(limit)*candleMs, 10))

	case params.StartTime != nil && params.EndTime != nil:
		q.Set("start", strconv.FormatInt(params.StartTime.UnixMilli(), 10))
		q.Set("end", strconv.FormatInt(params.EndTime.UnixMilli(), 10))
	}

	q.Set("limit", strconv.Itoa(limit))
	u.RawQuery = q.Encode()

	req := &rt.Request{
		Method: "GET",
		URL:    u.String(),
	}

	var response KlineResponse
	_, err = c.requester.Send(req, &response)
	if err != nil {
		c.log.ErrorWithData("Failed to fetch OHLCV from Bybit", map[string]any{
			"error":    err.Error(),
			"endpoint": u.String(),
		})
		return nil, fmt.Errorf("failed to fetch OHLCV from Bybit: %w", err)
	}

	if !response.OK() {
		return nil, fmt.Errorf("bybit kline returned retCode=%d retMsg=%s", response.RetCode, response.RetMsg)
	}

	// Bybit returns candles newest-first — reverse to chronological order
	list := response.Result.List
	records := make([]tt.OHLCVRecord, 0, len(list))

	for i := len(list) - 1; i >= 0; i-- {
		row := list[i]
		if len(row) < 6 {
			continue
		}

		openTimeMs, parseErr := strconv.ParseInt(row[0], 10, 64)
		if parseErr != nil {
			continue
		}

		records = append(records, tt.OHLCVRecord{
			OpenTime: openTimeMs / 1000, // ms → seconds
			Open:     row[1],
			High:     row[2],
			Low:      row[3],
			Close:    row[4],
			Volume:   row[5],
			IsClosed: true,
		})

		if limit > 0 && len(records) >= limit {
			break
		}
	}

	return records, nil
}
