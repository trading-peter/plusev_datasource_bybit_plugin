package bybit

import "encoding/json"


const (
	// BybitMaxLimit is the maximum number of candles per kline request
	BybitMaxLimit = 1000
	// DefaultLimit is the default number of candles when limit is not specified
	DefaultLimit = 200
)


// ── Generic response envelope ────────────────────────────────────────────────

// BybitResponse is the standard V5 API response envelope.
// retCode 0 = success; any non-zero value is an error.
type BybitResponse struct {
	RetCode int    `json:"retCode"`
	RetMsg  string `json:"retMsg"`
	Time    int64  `json:"time"`
}

func (r BybitResponse) OK() bool {
	return r.RetCode == 0
}

func (r BybitResponse) Error() string {
	return r.RetMsg
}


// ── Instruments Info (market discovery) ──────────────────────────────────────

type InstrumentsInfoResponse struct {
	BybitResponse
	Result struct {
		Category       string       `json:"category"`
		List           []Instrument `json:"list"`
		NextPageCursor string       `json:"nextPageCursor"`
	} `json:"result"`
}


type Instrument struct {
	Symbol         string         `json:"symbol"`
	BaseCoin       string         `json:"baseCoin"`
	QuoteCoin      string         `json:"quoteCoin"`
	ContractType   string         `json:"contractType"`
	Status         string         `json:"status"`
	PriceScale     string         `json:"priceScale"`
	PriceFilter    PriceFilter    `json:"priceFilter"`
	LotSizeFilter  LotSizeFilter  `json:"lotSizeFilter"`
	LeverageFilter *LeverageFilter `json:"leverageFilter,omitempty"`
	SettleCoin     string         `json:"settleCoin,omitempty"`
	FundingInterval int           `json:"fundingInterval,omitempty"`
}


type PriceFilter struct {
	MinPrice string `json:"minPrice"`
	MaxPrice string `json:"maxPrice"`
	TickSize string `json:"tickSize"`
}


// LotSizeFilter is a union of linear/inverse and spot fields.
// Linear/inverse: maxOrderQty, minOrderQty, qtyStep, maxMktOrderQty, minNotionalValue
// Spot: basePrecision, quotePrecision, minOrderAmt, maxLimitOrderQty, maxMarketOrderQty
type LotSizeFilter struct {
	MaxOrderQty      string `json:"maxOrderQty,omitempty"`
	MinOrderQty      string `json:"minOrderQty,omitempty"`
	QtyStep          string `json:"qtyStep,omitempty"`
	MaxMktOrderQty   string `json:"maxMktOrderQty,omitempty"`
	MinNotionalValue string `json:"minNotionalValue,omitempty"`

	BasePrecision    string `json:"basePrecision,omitempty"`
	QuotePrecision   string `json:"quotePrecision,omitempty"`
	MinOrderAmt      string `json:"minOrderAmt,omitempty"`
	MaxLimitOrderQty string `json:"maxLimitOrderQty,omitempty"`
	MaxMarketOrderQty string `json:"maxMarketOrderQty,omitempty"`
}


type LeverageFilter struct {
	MinLeverage    string `json:"minLeverage"`
	MaxLeverage    string `json:"maxLeverage"`
	LeverageStep   string `json:"leverageStep"`
}


// ── Kline (OHLCV) ────────────────────────────────────────────────────────────

type KlineResponse struct {
	BybitResponse
	Result struct {
		Symbol   string     `json:"symbol"`
		Category string     `json:"category"`
		// List is an array of string arrays, sorted newest-first.
		// Each element: [startTime, open, high, low, close, volume, turnover]
		List [][]string `json:"list"`
	} `json:"result"`
}


// ── Wallet Balance ───────────────────────────────────────────────────────────

type WalletBalanceResponse struct {
	BybitResponse
	Result struct {
		List []WalletAccount `json:"list"`
	} `json:"result"`
}


type WalletAccount struct {
	AccountType           string        `json:"accountType"`
	TotalEquity           string        `json:"totalEquity"`
	TotalWalletBalance    string        `json:"totalWalletBalance"`
	TotalMarginBalance    string        `json:"totalMarginBalance"`
	TotalAvailableBalance string        `json:"totalAvailableBalance"`
	TotalPerpUPL          string        `json:"totalPerpUPL"`
	TotalInitialMargin    string        `json:"totalInitialMargin"`
	TotalMaintenanceMargin string       `json:"totalMaintenanceMargin"`
	AccountIMRate         string        `json:"accountIMRate"`
	AccountMMRate         string        `json:"accountMMRate"`
	Coins                 []CoinBalance `json:"coin"`
}


type CoinBalance struct {
	Coin             string `json:"coin"`
	WalletBalance    string `json:"walletBalance"`
	Locked           string `json:"locked"`
	UsdValue         string `json:"usdValue"`
	AvailableToWithdraw string `json:"availableToWithdraw"`
	UnrealisedPnl    string `json:"unrealisedPnl"`
	CumRealisedPnl   string `json:"cumRealisedPnl"`
	BorrowAmount     string `json:"borrowAmount"`
	SpotBorrow       string `json:"spotBorrow"`
	TotalOrderIM     string `json:"totalOrderIM"`
	TotalPositionIM  string `json:"totalPositionIM"`
	TotalPositionMM  string `json:"totalPositionMM"`
}


// ── Position List ────────────────────────────────────────────────────────────

type PositionListResponse struct {
	BybitResponse
	Result struct {
		Category string     `json:"category"`
		List     []Position `json:"list"`
	} `json:"result"`
}


type Position struct {
	Symbol         string `json:"symbol"`
	Side           string `json:"side"` // "Buy", "Sell", ""
	Size           string `json:"size"`
	AvgPrice       string `json:"avgPrice"`
	MarkPrice      string `json:"markPrice"`
	UnrealisedPnl  string `json:"unrealisedPnl"`
	Leverage       string `json:"leverage"`
	PositionIdx    int    `json:"positionIdx"`
	LiqPrice       string `json:"liqPrice"`
	PositionStatus string `json:"positionStatus"`
	PositionValue  string `json:"positionValue"`
	PositionIM     string `json:"positionIM"`
	PositionMM     string `json:"positionMM"`
	StopLoss       string `json:"stopLoss"`
	TakeProfit     string `json:"takeProfit"`
	TrailingStop   string `json:"trailingStop"`
	CreatedTime    string `json:"createdTime"`
	UpdatedTime    string `json:"updatedTime"`
	IsReduceOnly   bool   `json:"isReduceOnly"`
}


// ── Order Realtime ───────────────────────────────────────────────────────────

type OrderRealtimeResponse struct {
	BybitResponse
	Result struct {
		Category string  `json:"category"`
		List     []Order `json:"list"`
	} `json:"result"`
}


type Order struct {
	OrderID       string `json:"orderId"`
	OrderLinkId   string `json:"orderLinkId"`
	Symbol        string `json:"symbol"`
	Side          string `json:"side"` // "Buy", "Sell"
	OrderType     string `json:"orderType"` // "Market", "Limit"
	Price         string `json:"price"`
	Qty           string `json:"qty"`
	CumExecQty    string `json:"cumExecQty"`
	CumExecValue  string `json:"cumExecValue"`
	OrderStatus   string `json:"orderStatus"`
	ReduceOnly    bool   `json:"reduceOnly"`
	TimeInForce   string `json:"timeInForce"`
	CreatedTime   string `json:"createdTime"`
	UpdatedTime   string `json:"updatedTime"`
	AvgPrice      string `json:"avgPrice"`
	LeavesQty     string `json:"leavesQty"`
	PositionIdx   int    `json:"positionIdx"`
	StopOrderType string `json:"stopOrderType"`
	TriggerPrice  string `json:"triggerPrice"`
	TakeProfit    string `json:"takeProfit"`
	StopLoss      string `json:"stopLoss"`
	CloseOnTrigger bool  `json:"closeOnTrigger"`
}


// ── Create Order ─────────────────────────────────────────────────────────────

type CreateOrderRequest struct {
	Category          string `json:"category"`
	Symbol            string `json:"symbol"`
	Side              string `json:"side"`
	OrderType         string `json:"orderType"`
	Qty               string `json:"qty"`
	Price             string `json:"price,omitempty"`
	OrderLinkId       string `json:"orderLinkId,omitempty"`
	TimeInForce       string `json:"timeInForce,omitempty"`
	ReduceOnly        bool   `json:"reduceOnly,omitempty"`
	TakeProfit        string `json:"takeProfit,omitempty"`
	StopLoss          string `json:"stopLoss,omitempty"`
	TpTriggerBy       string `json:"tpTriggerBy,omitempty"`
	SlTriggerBy       string `json:"slTriggerBy,omitempty"`
	PositionIdx       int    `json:"positionIdx,omitempty"`
	TriggerPrice      string `json:"triggerPrice,omitempty"`
	TriggerDirection  int    `json:"triggerDirection,omitempty"`
	TriggerBy         string `json:"triggerBy,omitempty"`
	CloseOnTrigger    bool   `json:"closeOnTrigger,omitempty"`
	MarketUnit        string `json:"marketUnit,omitempty"`
}


type CreateOrderResponse struct {
	BybitResponse
	Result struct {
		OrderID     string `json:"orderId"`
		OrderLinkId string `json:"orderLinkId"`
	} `json:"result"`
}


// ── Cancel Order ─────────────────────────────────────────────────────────────

type CancelOrderRequest struct {
	Category    string `json:"category"`
	Symbol      string `json:"symbol"`
	OrderID     string `json:"orderId,omitempty"`
	OrderLinkId string `json:"orderLinkId,omitempty"`
}


type CancelOrderResponse struct {
	BybitResponse
	Result struct {
		OrderID     string `json:"orderId"`
		OrderLinkId string `json:"orderLinkId"`
	} `json:"result"`
}


type CancelAllResponse struct {
	BybitResponse
	Result struct {
		SuccessCount string `json:"successCount"`
	} `json:"result"`
}


// ── WebSocket types ──────────────────────────────────────────────────────────

type WSSubscribeMessage struct {
	Op   string   `json:"op"`
	Args []string `json:"args"`
}


type WSKlineData struct {
	Start     int64  `json:"start"`
	End       int64  `json:"end"`
	Interval  string `json:"interval"`
	Open      string `json:"open"`
	High      string `json:"high"`
	Low       string `json:"low"`
	Close     string `json:"close"`
	Volume    string `json:"volume"`
	Turnover  string `json:"turnover"`
	Confirm   bool   `json:"confirm"`
	Timestamp int64  `json:"timestamp"`
}


type WSKlineMessage struct {
	Topic string        `json:"topic"`
	Type  string        `json:"type"`
	Data  []WSKlineData `json:"data"`
	Ts    int64         `json:"ts"`
}


type WSOperationResponse struct {
	Op      string `json:"op"`
	Success bool   `json:"success"`
	RetMsg  string `json:"ret_msg"`
	ConnID  string `json:"conn_id"`
}


// RawMessage is used for flexible JSON parsing of unknown messages
type RawMessage = json.RawMessage
