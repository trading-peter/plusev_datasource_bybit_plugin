package main

import (
	"strings"

	ex "github.com/plusev-terminal/go-plugin-common/datasrc/exchange"
	"github.com/plusev-terminal/go-plugin-common/plugin"
	commonstream "github.com/plusev-terminal/go-plugin-common/stream"
)


// handleOHLCVStream sets up a WebSocket stream for OHLCV data.
// Bybit public kline WS: wss://stream.bybit.com/v5/public/{category}
// No authentication required for public streams.
func (p *BybitPlugin) handleOHLCVStream(params map[string]any) plugin.Response {
	streamParams := ex.OHLCVStreamParamsFromMap(params)
	if strings.TrimSpace(streamParams.Market.Symbol) == "" {
		return plugin.ErrorResponseMsg("market.symbol is required")
	}

	assetType := strings.TrimSpace(streamParams.Market.AssetType)

	streamReq := plugin.StreamSetupRequest{
		StreamID:   "bybit_ohlcv_" + streamParams.Market.Symbol + "_" + streamParams.Timeframe,
		StreamType: "ohlcv",
		Parameters: map[string]any{
			"market":    streamParams.Market,
			"timeframe": streamParams.Timeframe,
			"private":   false,
			"streamContext": map[string]any{
				"symbol":    streamParams.Market.Symbol,
				"timeframe": streamParams.Timeframe,
				"assetType": assetType,
			},
		},
	}

	setupResp, err := p.client.PrepareStream(streamReq)
	if err != nil {
		log.ErrorWithData("handleOHLCVStream PrepareStream failed", map[string]any{
			"error": err,
		})
		return plugin.ErrorResponse(err)
	}

	if !setupResp.Success {
		log.ErrorWithData("handleOHLCVStream setup response failed", map[string]any{
			"error": setupResp.Error,
		})
		return plugin.ErrorResponseMsg(setupResp.Error)
	}

	marker := commonstream.StreamMarker{
		Stream:          true,
		StreamID:        streamReq.StreamID,
		WebSocketURL:    setupResp.WebSocketURL,
		Headers:         setupResp.Headers,
		Subprotocol:     setupResp.Subprotocol,
		InitialMessages: setupResp.InitialMessages,
		StreamContext:   setupResp.StreamContext,
		Heartbeat: &commonstream.StreamHeartbeatSpec{
			App: &commonstream.AppHeartbeatSpec{
				MatchJSONField:       "op",
				PingValue:            "ping",
				PongValue:            "pong",
				ClientPingIntervalMs: 20000,
			},
		},
	}

	if err := marker.Validate(); err != nil {
		return plugin.ErrorResponseMsg(err.Error())
	}

	if marker.Heartbeat != nil && marker.Heartbeat.App != nil {
		if err := marker.Heartbeat.App.Validate(); err != nil {
			return plugin.ErrorResponseMsg(err.Error())
		}
	}

	return plugin.SuccessTypedResponse("StreamMarker", marker)
}
