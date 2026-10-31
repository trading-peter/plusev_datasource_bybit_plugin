package main

import (
	"time"

	ex "github.com/plusev-terminal/go-plugin-common/datasrc/exchange"
	"github.com/plusev-terminal/go-plugin-common/plugin"
)


// handleGetOHLCV returns historical OHLCV data
func (p *BybitPlugin) handleGetOHLCV(params map[string]any) plugin.Response {
	if p.client == nil {
		return plugin.ErrorResponseMsg("client not initialized")
	}

	ohlcvParams := ex.GetOHLCVParamsFromMap(params)

	records, err := p.client.GetOHLCV(ohlcvParams)
	if err != nil {
		log.ErrorWithData("handleGetOHLCV error", map[string]any{
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
