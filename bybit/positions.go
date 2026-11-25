package bybit

import (
	"fmt"
	"net/url"
	"strconv"

	rt "github.com/plusev-terminal/go-plugin-common/v2/requester/types"
)


// bybitPositionListLimit is the largest page /v5/position/list serves.
const bybitPositionListLimit = 200

// linearSettleCoins are the settle coins of the linear category this plugin
// publishes: USDT perpetuals/futures and USDC perpetuals.
var linearSettleCoins = [...]string{"USDT", "USDC"}

// GetPositions returns every open linear position of the account, merged
// across the linear settle coins and across result pages.
//
// Bybit rejects a linear position/list that carries neither symbol nor
// settleCoin (retCode 10001), so an account-wide snapshot is one paged query
// per settle coin. Any failed query fails the whole call: a partial snapshot
// would show the positions of the other settle coin as flat.
func (c *Client) GetPositions() (PositionListResponse, error) {
	merged := PositionListResponse{}

	for _, coin := range linearSettleCoins {
		cursor := ""

		for {
			page, err := c.getPositionsPage(coin, cursor)
			if err != nil {
				return PositionListResponse{}, fmt.Errorf("settleCoin %s: %w", coin, err)
			}

			merged.RetCode, merged.RetMsg, merged.Time = page.RetCode, page.RetMsg, page.Time
			merged.Result.Category = page.Result.Category
			merged.Result.List = append(merged.Result.List, page.Result.List...)

			cursor = page.Result.NextPageCursor
			if cursor == "" {
				break
			}
		}
	}

	return merged, nil
}

// getPositionsPage fetches one page of the open linear positions settled in coin.
func (c *Client) getPositionsPage(coin, cursor string) (PositionListResponse, error) {
	query := url.Values{}
	query.Set("category", "linear")
	query.Set("settleCoin", coin)
	query.Set("limit", strconv.Itoa(bybitPositionListLimit))
	if cursor != "" {
		query.Set("cursor", cursor)
	}

	req := &rt.Request{
		Method: "GET",
		URL:    c.baseURL + "/v5/position/list?" + query.Encode(),
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
	}

	c.addAuthHeaders(req, "")

	var resp PositionListResponse
	_, err := c.requester.Send(req, &resp)
	if err != nil {
		return PositionListResponse{}, fmt.Errorf("failed to fetch positions: %w", err)
	}

	if !resp.OK() {
		return resp, fmt.Errorf("bybit position/list returned retCode=%d retMsg=%s", resp.RetCode, resp.RetMsg)
	}

	return resp, nil
}
