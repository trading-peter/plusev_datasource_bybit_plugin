package bybit

import (
	"fmt"
	"net/url"
	"strconv"

	"github.com/plusev-terminal/go-plugin-common/v2/datasrc/exchange"
	rt "github.com/plusev-terminal/go-plugin-common/v2/requester/types"
	tt "github.com/plusev-terminal/go-plugin-common/v2/trading"
)


// GetRecentOHLCV fetches the most recent N candles from Bybit.
// Simpler variant — no start/end time support, just fetches latest candles.
func (c *Client) GetRecentOHLCV(params exchange.GetOHLCVParams) ([]tt.OHLCVRecord, error) {
	symbol := params.Market.Symbol
	if symbol == "" {
		return nil, fmt.Errorf("market.symbol is required")
	}

	category := AssetTypeToCategory(params.Market.AssetType)

	interval, err := c.convertTimeframeToBybitInterval(params.Timeframe)
	if err != nil {
		return nil, err
	}

	limit := params.Limit
	if limit <= 0 {
		limit = DefaultLimit
	}
	if limit > BybitMaxLimit {
		limit = BybitMaxLimit
	}

	u, err := url.Parse(c.baseURL + "/v5/market/kline")
	if err != nil {
		return nil, fmt.Errorf("failed to parse kline url: %w", err)
	}

	q := u.Query()
	q.Set("category", category)
	q.Set("symbol", symbol)
	q.Set("interval", interval)
	q.Set("limit", strconv.Itoa(limit))
	u.RawQuery = q.Encode()

	req := &rt.Request{
		Method: "GET",
		URL:    u.String(),
	}

	var response KlineResponse
	_, err = c.requester.Send(req, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch recent OHLCV from Bybit: %w", err)
	}

	if !response.OK() {
		return nil, fmt.Errorf("bybit kline returned retCode=%d retMsg=%s", response.RetCode, response.RetMsg)
	}

	// Reverse newest-first to chronological order
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
			OpenTime: openTimeMs / 1000,
			Open:     row[1],
			High:     row[2],
			Low:      row[3],
			Close:    row[4],
			Volume:   row[5],
			IsClosed: true,
		})
	}

	return records, nil
}
