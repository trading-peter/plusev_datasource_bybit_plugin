package main

import (
	"time"

	"github.com/plusev-terminal/go-plugin-common/plugin"
)


// handleGetMarkets returns available trading pairs
func (p *BybitPlugin) handleGetMarkets(params map[string]any) plugin.Response {
	markets, err := p.client.GetMarkets()
	if err != nil {
		return plugin.ErrorResponse(err)
	}

	return plugin.SuccessResponse(markets, time.Hour*12)
}
