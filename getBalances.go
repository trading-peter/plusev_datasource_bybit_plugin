package main

import (
	"strings"
	"time"

	acct "github.com/plusev-terminal/go-plugin-common/datasrc/exchange/account"
	"github.com/plusev-terminal/go-plugin-common/plugin"
)


// handleGetBalances returns balances only (no account metadata beyond balances snapshot).
func (p *BybitPlugin) handleGetBalances(params map[string]any) plugin.Response {
	if p.client == nil {
		return plugin.ErrorResponseMsg("client not initialized")
	}

	walletResp, err := p.client.GetWalletBalance()
	if err != nil {
		return plugin.ErrorResponse(err)
	}

	fetchedAt := time.Now()
	if walletResp.Time > 0 {
		fetchedAt = time.UnixMilli(walletResp.Time)
	}

	resp := acct.BalancesResponse{
		FetchedAt: fetchedAt,
		Scopes:    map[acct.ScopeType]acct.BalanceScope{},
		Extra:     map[string]any{},
	}

	if len(walletResp.Result.List) > 0 {
		wa := walletResp.Result.List[0]

		scopeType := acct.ScopeUnified
		if strings.ToUpper(wa.AccountType) == "SPOT" {
			scopeType = acct.ScopeSpot
		}

		scope := acct.BalanceScope{
			Balances: map[string]acct.AssetBalance{},
			State:    &acct.ScopeState{},
			Extra:    map[string]any{},
		}

		for _, coin := range wa.Coins {
			asset := strings.ToUpper(strings.TrimSpace(coin.Coin))
			if asset == "" {
				continue
			}

			scope.Balances[asset] = acct.AssetBalance{
				Asset:            asset,
				Total:            coin.WalletBalance,
				AvailableToTrade: coin.AvailableToWithdraw,
				AvailableToWithdraw: coin.AvailableToWithdraw,
				Components: map[string]string{
					"locked":            coin.Locked,
					"unrealised_pnl":    coin.UnrealisedPnl,
					"cum_realised_pnl":  coin.CumRealisedPnl,
					"total_order_im":    coin.TotalOrderIM,
					"total_position_im": coin.TotalPositionIM,
					"total_position_mm": coin.TotalPositionMM,
				},
			}
		}

		scope.State.Equity = wa.TotalEquity
		scope.State.AvailableMargin = wa.TotalAvailableBalance
		scope.State.UsedMargin = wa.TotalInitialMargin
		scope.State.MaintenanceMargin = wa.TotalMaintenanceMargin
		scope.State.UnrealizedPnL = wa.TotalPerpUPL

		resp.Scopes[scopeType] = scope
	}

	return plugin.SuccessResponse(resp, time.Second*3)
}
