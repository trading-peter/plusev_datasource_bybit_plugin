package main

import (
	"reflect"
	"testing"

	m "github.com/plusev-terminal/go-plugin-common/meta"
)

// TestBybitManifestRateLimits pins the manifest rate-limit table so the
// declared budgets (hosts, limits, windows, scopes, paths, reserve) cannot
// silently drift. Values verified against
// https://bybit-exchange.github.io/docs/v5/rate-limit (retrieved 2026-10-31).
func TestBybitManifestRateLimits(t *testing.T) {
	targets := (&BybitPlugin{}).GetMeta().Resources.AllowedNetworkTargets

	limited := map[string]*m.NetworkTargetRule{}
	for i := range targets {
		pattern := targets[i].Pattern
		switch pattern {
		case "https://api.bybit.com/*", "https://api-testnet.bybit.com/*":
			limited[pattern] = &targets[i]
		default:
			if len(targets[i].Limits) > 0 {
				t.Errorf("%s: expected no rate limits on unused/stream host, got %#v", pattern, targets[i].Limits)
			}
			if targets[i].TradingReserve != nil {
				t.Errorf("%s: expected no trading reserve on unused/stream host", pattern)
			}
		}
	}
	if len(limited) != 2 {
		t.Fatalf("expected limits on exactly the 2 REST API hosts, got %d", len(limited))
	}

	want := []m.RateLimitRule{
		{ID: "ip-requests", Label: "Requests per IP", Limit: 600, WindowMs: 5000},
		{ID: "order-create", Label: "Create orders (per UID)", Limit: 10, WindowMs: 1000, Scope: m.RateLimitScopeCredential, Paths: []string{"/v5/order/create"}},
		{ID: "order-cancel", Label: "Cancel order (per UID)", Limit: 10, WindowMs: 1000, Scope: m.RateLimitScopeCredential, Paths: []string{"/v5/order/cancel"}},
		{ID: "order-cancel-all", Label: "Cancel all orders (per UID)", Limit: 10, WindowMs: 1000, Scope: m.RateLimitScopeCredential, Paths: []string{"/v5/order/cancel-all"}},
		{ID: "open-orders", Label: "Open orders (per UID)", Limit: 50, WindowMs: 1000, Scope: m.RateLimitScopeCredential, Paths: []string{"/v5/order/realtime"}},
		{ID: "positions", Label: "Positions (per UID)", Limit: 50, WindowMs: 1000, Scope: m.RateLimitScopeCredential, Paths: []string{"/v5/position/list"}},
		{ID: "wallet-balance", Label: "Wallet balance (per UID)", Limit: 50, WindowMs: 1000, Scope: m.RateLimitScopeCredential, Paths: []string{"/v5/account/wallet-balance"}},
	}

	for pattern, target := range limited {
		if !reflect.DeepEqual(target.Limits, want) {
			t.Errorf("%s limits mismatch:\n got: %#v\nwant: %#v", pattern, target.Limits, want)
		}
		if target.TradingReserve == nil || *target.TradingReserve != 0.3 {
			t.Errorf("%s: expected trading reserve 0.3, got %v", pattern, target.TradingReserve)
		}
	}
}

// TestGetRateLimitsOptsOutOfLegacyLayer pins the explicit legacy opt-out: the
// plugin declares its budgets as manifest limits, so GetRateLimits must return
// an empty (non-nil) slice to avoid double throttling.
func TestGetRateLimitsOptsOutOfLegacyLayer(t *testing.T) {
	got := (&BybitPlugin{}).GetRateLimits()
	if got == nil || len(got) != 0 {
		t.Errorf("expected empty non-nil legacy rate limits, got %#v", got)
	}
}
