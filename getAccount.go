package main

import (
	"strings"
	"time"

	acct "github.com/plusev-terminal/go-plugin-common/v2/datasrc/exchange/account"
	"github.com/plusev-terminal/go-plugin-common/v2/plugin"
)


// handleGetAccount returns normalized account details and balances.
func (p *BybitPlugin) handleGetAccount(params map[string]any) plugin.Response {
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

	account := acct.Account{
		Exchange:     "bybit",
		Type:         acct.AccountTypeFutures,
		FetchedAt:    fetchedAt,
		CustodyModel: acct.CustodyModelExchangeCustody,
		IsCustodial:  true,
		Scopes:       map[acct.ScopeType]acct.BalanceScope{},
		Extra:        map[string]any{},
	}

	if len(walletResp.Result.List) > 0 {
		wa := walletResp.Result.List[0]

		account.AccountID = wa.AccountType
		account.Extra["accountType"] = wa.AccountType
		account.Extra["totalEquity"] = wa.TotalEquity
		account.Extra["totalWalletBalance"] = wa.TotalWalletBalance

		scopeType := acct.ScopeUnified
		if strings.ToUpper(wa.AccountType) == "SPOT" {
			scopeType = acct.ScopeSpot
			account.Type = acct.AccountTypeSpot
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
				Components: map[string]string{
					"locked":            coin.Locked,
					"unrealised_pnl":    coin.UnrealisedPnl,
					"cum_realised_pnl":  coin.CumRealisedPnl,
					"total_order_im":    coin.TotalOrderIM,
					"total_position_im": coin.TotalPositionIM,
					"total_position_mm": coin.TotalPositionMM,
				},
				Extra: map[string]any{
					"usd_value":     coin.UsdValue,
					"borrow_amount": coin.BorrowAmount,
				},
			}
		}

		scope.State.Equity = wa.TotalEquity
		scope.State.AvailableMargin = wa.TotalAvailableBalance
		scope.State.UsedMargin = wa.TotalInitialMargin
		scope.State.MaintenanceMargin = wa.TotalMaintenanceMargin
		scope.State.UnrealizedPnL = wa.TotalPerpUPL
		scope.State.Extra = map[string]any{
			"account_im_rate": wa.AccountIMRate,
			"account_mm_rate": wa.AccountMMRate,
		}

		account.Scopes[scopeType] = scope
	}

	return plugin.SuccessResponse(account, time.Second*5)
}
