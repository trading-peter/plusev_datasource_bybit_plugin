package main

import (
	"fmt"
	"time"

	ex "github.com/plusev-terminal/go-plugin-common/v2/datasrc/exchange"
	acct "github.com/plusev-terminal/go-plugin-common/v2/datasrc/exchange/account"
	"github.com/plusev-terminal/go-plugin-common/v2/logging"
	m "github.com/plusev-terminal/go-plugin-common/v2/meta"
	"github.com/plusev-terminal/go-plugin-common/v2/plugin"
	"github.com/plusev-terminal/go-plugin-common/v2/requester"
	tt "github.com/plusev-terminal/go-plugin-common/v2/trading"
	"github.com/trading-peter/plusev_datasource_bybit_plugin/bybit"
)

// ============================================================================
// Main - Register the plugin
// ============================================================================

func init() {
	p := &BybitPlugin{}

	plugin.RegisterPlugin(p)

	plugin.RegisterStreamHandler(&streamHandlerWrapper{p: p})
}

var log = logging.NewLogger("bybit-datasource")

func main() {
	// Required for WASM, but can be empty
}

// ============================================================================
// Plugin Implementation
// ============================================================================

// BybitPlugin implements the DataSourcePlugin interface
type BybitPlugin struct {
	config *plugin.ConfigStore
	client *bybit.Client
}

// GetMeta returns the plugin metadata
func (p *BybitPlugin) GetMeta() m.Meta {
	return m.Meta{
		PluginID:    "bybit-datasource",
		Name:        "Bybit",
		AppID:       "datasrc",
		Category:    "cex",
		Description: "Bybit V5 data source — spot, USDT perpetuals, and USDT futures markets",
		Author:      "trading_peter",
		Version: "v1.1.0",
		Repository:  "https://github.com/trading-peter/plusev_datasource_bybit_plugin",
		Tags:        []string{"bybit", "crypto", "exchange", "perpetual", "spot", "futures"},
		Contacts: []m.AuthorContact{
			{
				Kind:  "x.com",
				Value: "https://x.com/trading_peter",
			},
		},
		Resources: m.ResourceAccess{
			AllowedNetworkTargets: []m.NetworkTargetRule{
				{Pattern: "https://api.bybit.com/*", Limits: bybitAPILimits(), TradingReserve: &bybitTradingReserve},
				{Pattern: "https://api-testnet.bybit.com/*", Limits: bybitAPILimits(), TradingReserve: &bybitTradingReserve},
				{Pattern: "https://api.bytick.com/*"},
				{Pattern: "wss://stream.bybit.com/*"},
				{Pattern: "wss://stream-testnet.bybit.com/*"},
			},
		},
		Features: []string{},
	}
}

// bybitTradingReserve keeps 30% of each API host's budget for order commands,
// which share the host with data pulls (wallet, positions, orders, klines).
var bybitTradingReserve = 0.3

// bybitAPILimits returns the Bybit v5 REST budgets for api.bybit.com; the
// testnet host applies the same limits.
//
// Sources (retrieved 2026-10-31):
//   - https://bybit-exchange.github.io/docs/v5/rate-limit
//
// Bybit v5 publishes one global per-IP cap plus per-endpoint requests/s per
// UID. The only weighted budget in the current docs (batch endpoints consume
// requests × orders) covers batch endpoints this plugin does not call, so no
// rule is Weighted. Where docs give per-category limits, the lowest value
// (linear/inverse) is declared; users on higher tiers can override in
// Terminal's Rate Limits UI.
func bybitAPILimits() []m.RateLimitRule {
	return []m.RateLimitRule{
		// All api.bybit.com traffic incl. public endpoints: 600 requests per 5 s per IP.
		// https://bybit-exchange.github.io/docs/v5/rate-limit (IP Limit → HTTP IP limit)
		m.NewRateLimitRule("ip-requests", 600, 5*time.Second).WithLabel("Requests per IP"),

		// POST /v5/order/create: 10/s per UID (linear/inverse; spot allows 20/s).
		// https://bybit-exchange.github.io/docs/v5/rate-limit (API Rate Limit Table → Trade)
		m.NewRateLimitRule("order-create", 10, time.Second).WithLabel("Create orders (per UID)").PerCredential().ForPaths("/v5/order/create"),

		// POST /v5/order/cancel: 10/s per UID (linear/inverse; spot allows 20/s).
		// https://bybit-exchange.github.io/docs/v5/rate-limit (API Rate Limit Table → Trade)
		m.NewRateLimitRule("order-cancel", 10, time.Second).WithLabel("Cancel order (per UID)").PerCredential().ForPaths("/v5/order/cancel"),

		// POST /v5/order/cancel-all: 10/s per UID (linear/inverse; option 1/s, spot 20/s).
		// https://bybit-exchange.github.io/docs/v5/rate-limit (API Rate Limit Table → Trade)
		m.NewRateLimitRule("order-cancel-all", 10, time.Second).WithLabel("Cancel all orders (per UID)").PerCredential().ForPaths("/v5/order/cancel-all"),

		// GET /v5/order/realtime: 50/s per UID.
		// https://bybit-exchange.github.io/docs/v5/rate-limit (API Rate Limit Table → Trade)
		m.NewRateLimitRule("open-orders", 50, time.Second).WithLabel("Open orders (per UID)").PerCredential().ForPaths("/v5/order/realtime"),

		// GET /v5/position/list: 50/s per UID.
		// https://bybit-exchange.github.io/docs/v5/rate-limit (API Rate Limit Table → Position)
		m.NewRateLimitRule("positions", 50, time.Second).WithLabel("Positions (per UID)").PerCredential().ForPaths("/v5/position/list"),

		// GET /v5/account/wallet-balance: 50/s per UID (accountType=UNIFIED).
		// https://bybit-exchange.github.io/docs/v5/rate-limit (API Rate Limit Table → Account)
		m.NewRateLimitRule("wallet-balance", 50, time.Second).WithLabel("Wallet balance (per UID)").PerCredential().ForPaths("/v5/account/wallet-balance"),
	}
}

// GetRateLimits intentionally returns an empty slice: the plugin's budgets are
// declared as manifest Limits on its network targets (see bybitAPILimits), and
// returning an empty array opts out of the legacy per-command limiter so the
// two layers don't double-throttle.
func (p *BybitPlugin) GetRateLimits() []plugin.RateLimit {
	return []plugin.RateLimit{}
}

// GetOrderCapabilities declares the order features the Bybit v5 create path
// really implements — keep in sync with buildBybitCreateOrderRequest; the
// host rejects any order feature not declared here.
//
//   - AssetTypes are the categories GetMarkets publishes: spot (category
//     spot) and perpetual/futures (both routed to category linear).
//   - Conditional orders send triggerPrice + triggerDirection, and
//     TriggerSource maps to v5 triggerBy (LastPrice/MarkPrice/IndexPrice).
//     An empty TriggerSource sends no triggerBy and Bybit applies its own
//     default, LastPrice.
//   - GTD stays undeclared: v5 has no GTD timeInForce and no expireTime
//     parameter. Trailing stops stay undeclared for the same reason —
//     /v5/order/create carries no trailing distance — as do ClosePosition,
//     PositionSide (PositionIdx is pinned to one-way mode) and every Extra
//     key (the create path reads none).
func (p *BybitPlugin) GetOrderCapabilities() ex.OrderCapabilities {
	return ex.OrderCapabilities{
		AssetTypes: []tt.AssetType{
			tt.AssetTypeSpot,
			tt.AssetTypePerpetual,
			tt.AssetTypeFutures,
		},
		OrderTypes: []acct.OrderType{
			acct.OrderTypeLimit,
			acct.OrderTypeMarket,
			acct.OrderTypeStop,
			acct.OrderTypeStopMarket,
			acct.OrderTypeStopLimit,
			acct.OrderTypeTakeProfit,
			acct.OrderTypeTakeProfitMarket,
			acct.OrderTypeTakeProfitLimit,
		},
		TimeInForce: []acct.TimeInForce{
			acct.TimeInForceGTC,
			acct.TimeInForceIOC,
			acct.TimeInForceFOK,
			acct.TimeInForcePostOnly,
			acct.TimeInForceGTX,
		},
		TriggerSources: []acct.TriggerSource{
			acct.TriggerSourceLast,
			acct.TriggerSourceMark,
			acct.TriggerSourceIndex,
		},
		DefaultTriggerSource: acct.TriggerSourceLast,
		ReduceOnly:           true,
		PostOnly:             true,
		QuoteQuantity:        true,
		ClientOrderID:        true,
	}
}

// GetConfigFields returns the configuration fields needed by this plugin
func (p *BybitPlugin) GetConfigFields() []plugin.ConfigField {
	if p.client == nil {
		p.client = bybit.NewClient(requester.NewRequester(), "https://api.bybit.com", false)
	}

	return p.client.GetConfigFields()
}

// OnInit is called when the plugin is initialized with user configuration
func (p *BybitPlugin) OnInit(config *plugin.ConfigStore) error {
	p.config = config

	isTestnet := config.GetBool("testnet")
	baseURL := "https://api.bybit.com"
	if isTestnet {
		baseURL = "https://api-testnet.bybit.com"
	}

	p.client = bybit.NewClient(requester.NewRequester(), baseURL, isTestnet)

	credentials := make(map[string]string)

	if apiKey := config.GetString("apiKey"); apiKey != "" {
		credentials["apiKey"] = apiKey
	}

	if secretKey := config.GetString("secretKey"); secretKey != "" {
		credentials["secretKey"] = secretKey
	}

	if accountType := config.GetString("accountType"); accountType != "" {
		credentials["accountType"] = accountType
	}

	p.client.SetCredentials(credentials)

	return nil
}

// OnShutdown is called when the plugin is being shut down
func (p *BybitPlugin) OnShutdown() error {
	return nil
}

// RegisterCommands registers all command handlers
func (p *BybitPlugin) RegisterCommands(router *plugin.CommandRouter) {
	router.Register(ex.CMD_GET_MARKETS, p.handleGetMarkets)
	router.Register(ex.CMD_GET_TIMEFRAMES, p.handleGetTimeframes)
	router.Register(ex.CMD_OHLCV_STREAM, p.handleOHLCVStream)
	router.Register(ex.CMD_GET_OHLCV, p.handleGetOHLCV)
	router.Register(ex.CMD_GET_ACCOUNT, p.handleGetAccount)
	router.Register(ex.CMD_GET_BALANCES, p.handleGetBalances)
	router.Register(ex.CMD_GET_POSITIONS, p.handleGetPositions)
	router.Register(ex.CMD_GET_ORDERS, p.handleGetOrders)
	router.Register(ex.CMD_CREATE_ORDERS, p.handleCreateOrders)
	router.Register(ex.CMD_CANCEL_ORDERS, p.handleCancelOrders)
	router.Register("getRecentOHLCV", p.handleGetRecentOHLCV)
}

// ============================================================================
// Stream Handler Wrapper
// ============================================================================

type streamHandlerWrapper struct {
	p *BybitPlugin
}

func (w *streamHandlerWrapper) HandleStreamMessage(req plugin.StreamMessageRequest) (plugin.StreamMessageResponse, error) {
	if w.p.client == nil {
		log.ErrorWithData("streamHandlerWrapper.HandleStreamMessage: client not initialized", map[string]any{})
		return plugin.StreamMessageResponse{Success: false, Action: "ignore"}, fmt.Errorf("client not initialized")
	}

	response, err := w.p.client.HandleStreamMessage(req)

	if err != nil {
		log.ErrorWithData("streamHandlerWrapper.HandleStreamMessage client response", map[string]any{
			"response": response,
			"error":    err,
		})
	}

	return response, err
}

func (w *streamHandlerWrapper) HandleConnectionEvent(event plugin.StreamConnectionEvent) (plugin.StreamConnectionResponse, error) {
	if w.p.client == nil {
		log.ErrorWithData("streamHandlerWrapper.HandleConnectionEvent: client not initialized", map[string]any{})
		return plugin.StreamConnectionResponse{Success: false, Action: "ignore"}, fmt.Errorf("client not initialized")
	}

	response, err := w.p.client.HandleConnectionEvent(event)

	if err != nil {
		log.ErrorWithData("streamHandlerWrapper.HandleConnectionEvent client response", map[string]any{
			"response": response,
			"error":    err,
		})
	}

	return response, err
}
