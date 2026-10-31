package bybit

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/plusev-terminal/go-plugin-common/plugin"
	tt "github.com/plusev-terminal/go-plugin-common/trading"
)


// PrepareStream prepares streaming connection setup for Bybit public kline WebSocket.
// Bybit public WS: wss://stream.bybit.com/v5/public/{category}
// Subscribe topic: kline.{interval}.{symbol}
func (c *Client) PrepareStream(request plugin.StreamSetupRequest) (plugin.StreamSetupResponse, error) {
	streamContext, _ := request.Parameters["streamContext"].(map[string]any)
	if streamContext == nil {
		return plugin.StreamSetupResponse{Success: false, Error: "streamContext is required"}, nil
	}

	symbol, _ := streamContext["symbol"].(string)
	timeframe, _ := streamContext["timeframe"].(string)
	assetType, _ := streamContext["assetType"].(string)

	if strings.TrimSpace(symbol) == "" || strings.TrimSpace(timeframe) == "" {
		return plugin.StreamSetupResponse{
			Success: false,
			Error:   "streamContext.symbol and streamContext.timeframe are required",
		}, nil
	}

	// Convert timeframe to Bybit interval
	interval, err := c.convertTimeframeToBybitInterval(timeframe)
	if err != nil {
		return plugin.StreamSetupResponse{Success: false, Error: err.Error()}, nil
	}

	// Determine category for the WS URL path
	category := AssetTypeToCategory(assetType)

	// Build WebSocket URL
	wsURL := fmt.Sprintf("%s/v5/public/%s", c.wsBaseURL, category)

	// Build subscribe topic: kline.{interval}.{symbol}
	topic := fmt.Sprintf("kline.%s.%s", interval, symbol)
	subscribeMsg := WSSubscribeMessage{
		Op:   "subscribe",
		Args: []string{topic},
	}

	msgBytes, err := json.Marshal(subscribeMsg)
	if err != nil {
		return plugin.StreamSetupResponse{
			Success: false,
			Error:   fmt.Sprintf("failed to marshal subscribe message: %v", err),
		}, nil
	}

	response := plugin.StreamSetupResponse{
		Success:         true,
		WebSocketURL:    wsURL,
		Headers:         nil,
		Subprotocol:     "",
		InitialMessages: []string{string(msgBytes)},
		StreamContext:   streamContext,
	}

	return response, nil
}


// HandleStreamMessage processes incoming WebSocket messages from Bybit.
func (c *Client) HandleStreamMessage(request plugin.StreamMessageRequest) (plugin.StreamMessageResponse, error) {
	// Ignore pong responses (host sends {"op":"ping"}, Bybit replies {"op":"pong"})
	var opResp WSOperationResponse
	if err := json.Unmarshal(request.Message, &opResp); err == nil {
		if opResp.Op == "pong" || opResp.Op == "ping" {
			return plugin.StreamMessageResponse{Success: true, Action: "ignore"}, nil
		}

		// Subscribe/unsubscribe confirmations
		if opResp.Op == "subscribe" || opResp.Op == "unsubscribe" {
			return plugin.StreamMessageResponse{Success: true, Action: "ignore"}, nil
		}
	}

	// Try to parse as kline message
	var klineMsg WSKlineMessage
	if err := json.Unmarshal(request.Message, &klineMsg); err == nil {
		if strings.HasPrefix(klineMsg.Topic, "kline.") && len(klineMsg.Data) > 0 {
			symbol, _ := request.StreamContext["symbol"].(string)
			timeframe, _ := request.StreamContext["timeframe"].(string)

			// Verify topic matches this stream
			interval, _ := c.convertTimeframeToBybitInterval(timeframe)
			expectedTopic := fmt.Sprintf("kline.%s.%s", interval, symbol)

			if klineMsg.Topic != expectedTopic {
				return plugin.StreamMessageResponse{Success: true, Action: "ignore"}, nil
			}

			data := klineMsg.Data[0]

			record := tt.OHLCVRecord{
				OpenTime: data.Start / 1000, // ms → seconds
				Open:     data.Open,
				High:     data.High,
				Low:      data.Low,
				Close:    data.Close,
				Volume:   data.Volume,
				IsClosed: data.Confirm,
			}

			return plugin.StreamMessageResponse{
				Success:  true,
				Action:   "data",
				DataType: "ohlcv",
				Data:     record,
			}, nil
		}
	}

	return plugin.StreamMessageResponse{Success: true, Action: "ignore"}, nil
}


// HandleConnectionEvent handles stream connection lifecycle events.
func (c *Client) HandleConnectionEvent(event plugin.StreamConnectionEvent) (plugin.StreamConnectionResponse, error) {
	switch event.EventType {
	case "disconnected":
		return plugin.StreamConnectionResponse{
			Success: true,
			Action:  "reconnect",
		}, nil
	case "error":
		c.log.ErrorWithData("WebSocket error", map[string]any{
			"streamID": event.StreamID,
			"error":    event.Error,
		})
	}

	return plugin.StreamConnectionResponse{
		Success: true,
		Action:  "ignore",
	}, nil
}
