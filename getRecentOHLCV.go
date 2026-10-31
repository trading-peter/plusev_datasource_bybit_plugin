package main

import (
	"time"

	ex "github.com/plusev-terminal/go-plugin-common/v2/datasrc/exchange"
	"github.com/plusev-terminal/go-plugin-common/v2/plugin"
)


// handleGetRecentOHLCV returns recent OHLCV data (no start/end time support)
func (p *BybitPlugin) handleGetRecentOHLCV(params map[string]any) plugin.Response {
	if p.client == nil {
		return plugin.ErrorResponseMsg("client not initialized")
	}

	ohlcvParams := ex.GetOHLCVParamsFromMap(params)

	records, err := p.client.GetRecentOHLCV(ohlcvParams)
	if err != nil {
		log.ErrorWithData("handleGetRecentOHLCV error", map[string]any{
			"error": err.Error(),
		})
		return plugin.ErrorResponse(err)
	}

	if ohlcvParams.CacheForSeconds > 0 {
		cacheDuration := time.Duration(ohlcvParams.CacheForSeconds) * time.Second
		return plugin.SuccessResponse(records, cacheDuration)
	}

	return plugin.SuccessResponse(records)
}
