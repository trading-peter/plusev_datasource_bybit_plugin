package bybit

import (
	"fmt"
	"strconv"
	"strings"

	rt "github.com/plusev-terminal/go-plugin-common/requester/types"
	tt "github.com/plusev-terminal/go-plugin-common/trading"
)


// AssetTypeToCategory maps Terminal's AssetType to Bybit's category parameter.
func AssetTypeToCategory(assetType string) string {
	switch strings.ToLower(strings.TrimSpace(assetType)) {
	case "spot":
		return "spot"
	case "perpetual", "futures", "linear":
		return "linear"
	case "inverse":
		return "inverse"
	case "option":
		return "option"
	default:
		return "linear"
	}
}


// GetMarkets returns all available trading markets from Bybit.
// Queries both spot and linear (USDT perps/futures) categories.
func (c *Client) GetMarkets() ([]tt.Market, error) {
	var markets []tt.Market

	// Fetch linear (USDT perps + USDC perps + USDT futures)
	linearMarkets, err := c.getCategoryMarkets("linear")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch linear markets: %w", err)
	}
	markets = append(markets, linearMarkets...)

	// Fetch spot
	spotMarkets, err := c.getCategoryMarkets("spot")
	if err != nil {
		c.log.ErrorWithData("Failed to fetch spot markets", map[string]any{"error": err})
	} else {
		markets = append(markets, spotMarkets...)
	}

	return markets, nil
}


// getCategoryMarkets fetches instruments-info for a single category,
// handling pagination for categories that support it (linear/inverse).
func (c *Client) getCategoryMarkets(category string) ([]tt.Market, error) {
	var markets []tt.Market
	var cursor string

	for {
		queryParams := fmt.Sprintf("category=%s", category)
		if cursor != "" {
			queryParams += "&cursor=" + cursor
		}

		req := &rt.Request{
			Method: "GET",
			URL:    c.baseURL + "/v5/market/instruments-info?" + queryParams,
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
		}

		var response InstrumentsInfoResponse
		_, err := c.requester.Send(req, &response)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch instruments-info for %s: %w", category, err)
		}

		if !response.OK() {
			return nil, fmt.Errorf("bybit instruments-info returned retCode=%d retMsg=%s", response.RetCode, response.RetMsg)
		}

		for _, inst := range response.Result.List {
			market := instrumentToMarket(inst, category)
			if market.Symbol != "" {
				markets = append(markets, market)
			}
		}

		// Spot does not support pagination
		if category == "spot" {
			break
		}

		cursor = strings.TrimSpace(response.Result.NextPageCursor)
		if cursor == "" {
			break
		}
	}

	return markets, nil
}


// instrumentToMarket converts a Bybit instrument to a normalized Market.
func instrumentToMarket(inst Instrument, category string) tt.Market {
	// Skip non-trading instruments
	status := strings.ToUpper(strings.TrimSpace(inst.Status))
	if status != "" && status != "TRADING" {
		return tt.Market{}
	}

	var assetType string
	var label string

	base := strings.ToUpper(inst.BaseCoin)
	quote := strings.ToUpper(inst.QuoteCoin)

	if category == "spot" {
		assetType = "spot"
		label = fmt.Sprintf("%s/%s", base, quote)
	} else {
		// Determine perpetual vs futures from contract type
		ct := strings.ToUpper(inst.ContractType)
		switch {
		case strings.Contains(ct, "PERPETUAL"):
			assetType = "perpetual"
		case strings.Contains(ct, "FUTURES"):
			assetType = "futures"
		default:
			assetType = "perpetual"
		}
		label = fmt.Sprintf("%s-PERP", base)
		if assetType == "futures" {
			label = fmt.Sprintf("%s-FUT", base)
		}
	}

	// Precision from tick size
	priceTick := strings.TrimSpace(inst.PriceFilter.TickSize)
	qtyTick := strings.TrimSpace(inst.LotSizeFilter.QtyStep)
	if qtyTick == "" {
		qtyTick = strings.TrimSpace(inst.LotSizeFilter.BasePrecision)
	}

	pricePrecision := countPrecisionFromTickString(priceTick)
	qtyPrecision := countPrecisionFromTickString(qtyTick)

	// Quantity limits
	minQty := strings.TrimSpace(inst.LotSizeFilter.MinOrderQty)
	if minQty == "" {
		minQty = strings.TrimSpace(inst.LotSizeFilter.MinOrderAmt)
	}
	maxQty := strings.TrimSpace(inst.LotSizeFilter.MaxOrderQty)
	if maxQty == "" {
		maxQty = strings.TrimSpace(inst.LotSizeFilter.MaxLimitOrderQty)
	}
	minNotional := strings.TrimSpace(inst.LotSizeFilter.MinNotionalValue)

	market := tt.Market{
		Label:            label,
		Symbol:           inst.Symbol,
		Base:             base,
		Quote:            quote,
		AssetType:        assetType,
		PriceTick:        priceTick,
		QuantityTick:     qtyTick,
		PricePrecision:   pricePrecision,
		QuantityPrecision: qtyPrecision,
		MinQuantity:      minQty,
		MaxQuantity:      maxQty,
		MinNotional:      minNotional,
		Status:           status,
	}

	// Leverage info (linear/inverse only)
	if inst.LeverageFilter != nil {
		market.MaxLeverage = strings.TrimSpace(inst.LeverageFilter.MaxLeverage)
	}

	// Funding interval (perps only, in minutes → hours)
	if inst.FundingInterval > 0 {
		market.FundingInterval = inst.FundingInterval / 60
	}

	// Contract size for inverse
	if strings.TrimSpace(inst.SettleCoin) != "" && category == "inverse" {
		market.ContractSize = "1"
	}

	return market
}


func countPrecisionFromTickString(tick string) int {
	if tick == "" {
		return 0
	}

	if strings.ContainsAny(tick, "eE") {
		if f, err := strconv.ParseFloat(tick, 64); err == nil {
			tick = strconv.FormatFloat(f, 'f', 18, 64)
		}
	}

	if tick == "0" || tick == "0.0" {
		return 0
	}

	idx := strings.Index(tick, ".")
	if idx == -1 {
		return 0
	}

	dec := strings.TrimRight(tick[idx+1:], "0")
	if dec == "" {
		return 0
	}

	if len(dec) > 12 {
		return 12
	}

	return len(dec)
}


// GetTimeframes returns the timeframes supported by Bybit.
func (c *Client) GetTimeframes() []tt.Timeframe {
	return []tt.Timeframe{
		{Value: 1, Unit: tt.Minutes},
		{Value: 3, Unit: tt.Minutes},
		{Value: 5, Unit: tt.Minutes},
		{Value: 15, Unit: tt.Minutes},
		{Value: 30, Unit: tt.Minutes},
		{Value: 1, Unit: tt.Hours},
		{Value: 2, Unit: tt.Hours},
		{Value: 4, Unit: tt.Hours},
		{Value: 6, Unit: tt.Hours},
		{Value: 12, Unit: tt.Hours},
		{Value: 1, Unit: tt.Days},
		{Value: 1, Unit: tt.Weeks},
		{Value: 1, Unit: tt.Months},
	}
}


// convertTimeframeToBybitInterval converts our timeframe format to Bybit's interval string.
// Bybit intervals: 1, 3, 5, 15, 30, 60, 120, 240, 360, 720, D, W, M
func (c *Client) convertTimeframeToBybitInterval(timeframe string) (string, error) {
	tf, err := tt.TimeframeFromString(timeframe)
	if err != nil {
		return "", fmt.Errorf("invalid timeframe format: %w", err)
	}

	switch tf.Unit {
	case tt.Minutes:
		return strconv.FormatUint(tf.Value, 10), nil
	case tt.Hours:
		return strconv.FormatUint(tf.Value*60, 10), nil
	case tt.Days:
		return "D", nil
	case tt.Weeks:
		return "W", nil
	case tt.Months:
		return "M", nil
	}

	return "", fmt.Errorf("unsupported timeframe unit: %s", tf.Unit)
}
