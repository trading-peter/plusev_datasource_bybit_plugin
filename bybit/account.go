package bybit

import (
	"fmt"

	rt "github.com/plusev-terminal/go-plugin-common/requester/types"
)


// GetWalletBalance queries /v5/account/wallet-balance for the configured account type.
func (c *Client) GetWalletBalance() (WalletBalanceResponse, error) {
	req := &rt.Request{
		Method: "GET",
		URL:    c.baseURL + "/v5/account/wallet-balance?accountType=" + c.accountType,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
	}

	c.addAuthHeaders(req, "")

	var resp WalletBalanceResponse
	_, err := c.requester.Send(req, &resp)
	if err != nil {
		return WalletBalanceResponse{}, fmt.Errorf("failed to fetch wallet-balance: %w", err)
	}

	if !resp.OK() {
		return resp, fmt.Errorf("bybit wallet-balance returned retCode=%d retMsg=%s", resp.RetCode, resp.RetMsg)
	}

	return resp, nil
}
