package bybit

import (
	"encoding/json"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/plusev-terminal/go-plugin-common/v2/datasrc/exchange"
	rt "github.com/plusev-terminal/go-plugin-common/v2/requester/types"
	tt "github.com/plusev-terminal/go-plugin-common/v2/trading"
)

// klineDoer answers /v5/market/kline the way the live API does (measured
// 2026-11-25): `end` is inclusive, a window narrower than `limit` candles is
// served whole, a wider one is answered with its newest `limit` candles, and
// a request without `end` gets the oldest `limit` candles from `start`. Rows
// come back newest first.
func klineDoer(t *testing.T) doerFunc {
	return func(req *rt.Request) string {
		q := mustQuery(t, req.URL, "/v5/market/kline")

		limit, err := strconv.Atoi(q.Get("limit"))
		if err != nil {
			t.Fatalf("kline limit %q: %v", q.Get("limit"), err)
		}

		intervalMs := int64(60_000)
		if minutes, err := strconv.ParseInt(q.Get("interval"), 10, 64); err == nil {
			intervalMs = minutes * 60_000
		}

		startMs, err := strconv.ParseInt(q.Get("start"), 10, 64)
		if err != nil {
			t.Fatalf("kline start %q: %v", q.Get("start"), err)
		}

		firstOpen := ((startMs + intervalMs - 1) / intervalMs) * intervalMs

		var opens []int64
		if endMs, err := strconv.ParseInt(q.Get("end"), 10, 64); err != nil {
			for i := int64(0); i < int64(limit); i++ {
				opens = append(opens, firstOpen+i*intervalMs)
			}
		} else {
			for open := (endMs / intervalMs) * intervalMs; open >= firstOpen && len(opens) < limit; open -= intervalMs {
				opens = append(opens, open)
			}
		}

		rows := make([][]string, len(opens))
		for i, open := range opens {
			rows[i] = []string{strconv.FormatInt(open, 10), "1", "2", "0", "1", "10", "10"}
		}

		rowsJSON, err := json.Marshal(rows)
		if err != nil {
			t.Fatalf("marshal kline rows: %v", err)
		}

		return fmt.Sprintf(`{"retCode":0,"retMsg":"OK","result":{"category":"linear","list":%s}}`, rowsJSON)
	}
}

// TestGetOHLCVServesOldestCandlesFromStart guards the history backfill bug:
// Bybit answers a window wider than `limit` candles with its newest ones, so
// market_proxy's gap fill fetched a single page and warm-up batches silently
// skipped their first candle. A start-anchored request must be served from
// its start.
func TestGetOHLCVServesOldestCandlesFromStart(t *testing.T) {
	start := time.Unix(1_700_000_160, 0) // minute-aligned
	withinLimit := time.Unix(start.Unix()+3*60, 0)
	widerThanLimit := time.Unix(start.Unix()+24*3600, 0)

	cases := []struct {
		name      string
		end       *time.Time
		limit     int
		wantCount int
		wantFirst int64
		wantLast  int64
	}{
		{"forward from start", nil, 1000, 1000, start.Unix(), start.Unix() + 999*60},
		{"window wider than limit", &widerThanLimit, 0, DefaultLimit, start.Unix(), start.Unix() + (DefaultLimit-1)*60},
		{"window within limit", &withinLimit, 0, 4, start.Unix(), withinLimit.Unix()},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client := NewClient(klineDoer(t), "https://api.bybit.com", false)

			records, err := client.GetOHLCV(exchange.GetOHLCVParams{
				Market:    tt.Market{Symbol: "ETHUSDT", AssetType: tt.AssetTypePerpetual},
				Timeframe: "1m",
				StartTime: &start,
				EndTime:   tc.end,
				Limit:     tc.limit,
			})
			if err != nil {
				t.Fatalf("GetOHLCV: %v", err)
			}

			if len(records) != tc.wantCount {
				t.Fatalf("got %d candles, want %d", len(records), tc.wantCount)
			}

			if first, last := records[0].OpenTime, records[len(records)-1].OpenTime; first != tc.wantFirst || last != tc.wantLast {
				t.Errorf("candles span [%d..%d], want [%d..%d]", first, last, tc.wantFirst, tc.wantLast)
			}
		})
	}
}
