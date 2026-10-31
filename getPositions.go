package main

import (
	"strconv"
	"strings"
	"time"

	acct "github.com/plusev-terminal/go-plugin-common/datasrc/exchange/account"
	"github.com/plusev-terminal/go-plugin-common/plugin"
)


// handleGetPositions returns open positions (linear/derivatives only).
func (p *BybitPlugin) handleGetPositions(params map[string]any) plugin.Response {
	if p.client == nil {
		return plugin.ErrorResponseMsg("client not initialized")
	}

	posResp, err := p.client.GetPositions()
	if err != nil {
		return plugin.ErrorResponse(err)
	}

	positions := make([]acct.Position, 0, len(posResp.Result.List))

	for _, row := range posResp.Result.List {
		side := normalizeBybitPositionSide(row.Side)

		// Skip empty positions
		if side == "" && strings.TrimSpace(row.Size) == "0" {
			continue
		}

		pos := acct.Position{
			Symbol:           row.Symbol,
			Side:             side,
			Quantity:         row.Size,
			EntryPrice:       row.AvgPrice,
			MarkPrice:        row.MarkPrice,
			UnrealizedPnL:    row.UnrealisedPnl,
			LiquidationPrice: row.LiqPrice,
			IsIsolated:       false,
			Components: map[string]string{
				"position_value":  row.PositionValue,
				"position_im":     row.PositionIM,
				"position_mm":     row.PositionMM,
				"stop_loss":       row.StopLoss,
				"take_profit":     row.TakeProfit,
				"trailing_stop":   row.TrailingStop,
				"position_status": row.PositionStatus,
			},
			Extra: map[string]any{
				"position_idx":   row.PositionIdx,
				"is_reduce_only": row.IsReduceOnly,
			},
		}

		if lev := strings.TrimSpace(row.Leverage); lev != "" {
			if levInt, parseErr := strconv.Atoi(lev); parseErr == nil {
				pos.Leverage = levInt
			}
		}

		positions = append(positions, pos)
	}

	fetchedAt := time.Now()
	if posResp.Time > 0 {
		fetchedAt = time.UnixMilli(posResp.Time)
	}

	resp := acct.PositionsResponse{
		FetchedAt: fetchedAt,
		Scopes: map[acct.ScopeType]acct.PositionScope{
			acct.ScopeFutures: {
				Positions: positions,
				Extra:     map[string]any{},
			},
		},
		Extra: map[string]any{},
	}

	return plugin.SuccessResponse(resp, time.Second*3)
}


func normalizeBybitPositionSide(side string) string {
	s := strings.ToUpper(strings.TrimSpace(side))
	switch s {
	case "BUY":
		return "long"
	case "SELL":
		return "short"
	}
	return ""
}
