package bybit

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	rt "github.com/plusev-terminal/go-plugin-common/v2/requester/types"
)

// One Bybit position row each, as /v5/position/list returns them.
const (
	usdtBTC = `{"symbol":"BTCUSDT","side":"Buy","size":"0.01","avgPrice":"60000"}`
	usdtETH = `{"symbol":"ETHUSDT","side":"Sell","size":"0.2","avgPrice":"3000"}`
	usdcETH = `{"symbol":"ETHPERP","side":"Sell","size":"1.5","avgPrice":"1700"}`
)

// positionsPage is a successful position/list answer.
func positionsPage(list, nextPageCursor string) string {
	return fmt.Sprintf(`{"retCode":0,"retMsg":"OK","result":{"category":"linear","list":[%s],"nextPageCursor":%q}}`, list, nextPageCursor)
}

// TestGetPositionsMergesSettleCoinsAndPages guards the account-monitor bug:
// Bybit rejects a linear position/list that carries neither symbol nor
// settleCoin (retCode 10001), so an account snapshot needs one paged query
// per linear settle coin — USDC perpetuals (ETHPERP) included.
func TestGetPositionsMergesSettleCoinsAndPages(t *testing.T) {
	pages := map[string]string{
		"USDT|":   positionsPage(usdtBTC, "p2"),
		"USDT|p2": positionsPage(usdtETH, ""),
		"USDC|":   positionsPage(usdcETH, ""),
	}

	client := NewClient(doerFunc(func(req *rt.Request) string {
		q := mustQuery(t, req.URL, "/v5/position/list")

		// Same validation as the live API: linear needs symbol or settleCoin.
		if q.Get("symbol") == "" && q.Get("settleCoin") == "" {
			return `{"retCode":10001,"retMsg":"Missing some parameters that must be filled in, symbol or settleCoin"}`
		}

		page, ok := pages[q.Get("settleCoin")+"|"+q.Get("cursor")]
		if !ok {
			t.Errorf("unexpected position/list query %s", req.URL)
			return `{"retCode":10001,"retMsg":"unexpected query"}`
		}

		return page
	}), "https://api.bybit.com", false)

	positions, err := client.GetPositions()
	if err != nil {
		t.Fatalf("GetPositions: %v", err)
	}

	symbols := make([]string, 0, len(positions.Result.List))
	for _, pos := range positions.Result.List {
		symbols = append(symbols, pos.Symbol)
	}

	if want := []string{"BTCUSDT", "ETHUSDT", "ETHPERP"}; !slices.Equal(symbols, want) {
		t.Errorf("positions = %v, want %v", symbols, want)
	}
}

// TestGetPositionsFailsClosedOnSettleCoinError: a partial snapshot would show
// the failed settle coin's positions as flat, so one rejected query fails the
// whole call instead of returning the rest.
func TestGetPositionsFailsClosedOnSettleCoinError(t *testing.T) {
	client := NewClient(doerFunc(func(req *rt.Request) string {
		q := mustQuery(t, req.URL, "/v5/position/list")

		if q.Get("settleCoin") == "USDC" {
			return `{"retCode":10005,"retMsg":"permission denied"}`
		}

		return positionsPage(usdtBTC, "")
	}), "https://api.bybit.com", false)

	positions, err := client.GetPositions()
	if err == nil {
		t.Fatal("GetPositions must fail when a settle coin query is rejected")
	}

	if len(positions.Result.List) != 0 {
		t.Errorf("failed snapshot returned %d positions, want none", len(positions.Result.List))
	}

	if !strings.Contains(err.Error(), "USDC") || !strings.Contains(err.Error(), "permission denied") {
		t.Errorf("error must name the failed settle coin and Bybit's message, got %q", err)
	}
}
