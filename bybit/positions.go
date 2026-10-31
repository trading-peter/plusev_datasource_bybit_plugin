package bybit

import (
	"fmt"

	rt "github.com/plusev-terminal/go-plugin-common/requester/types"
)


// GetPositions queries open positions for the linear category.
func (c *Client) GetPositions() (PositionListResponse, error) {
	req := &rt.Request{
		Method: "GET",
		URL:    c.baseURL + "/v5/position/list?category=linear",
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
