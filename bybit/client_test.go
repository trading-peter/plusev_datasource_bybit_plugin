package bybit

import (
	"encoding/json"
	"net/url"
	"testing"

	rt "github.com/plusev-terminal/go-plugin-common/v2/requester/types"
)

// doerFunc lets a test answer every request with a closure, so the URLs the
// client builds are observable without a network or the WASM host.
type doerFunc func(req *rt.Request) string

func (f doerFunc) Send(req *rt.Request, v any) (*rt.Response, error) {
	if err := json.Unmarshal([]byte(f(req)), v); err != nil {
		return nil, err
	}

	return &rt.Response{Status: 200}, nil
}

// mustQuery asserts the request targeted path and returns its query.
func mustQuery(t *testing.T, rawURL, path string) url.Values {
	t.Helper()

	u, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("parse request URL %s: %v", rawURL, err)
	}

	if u.Path != path {
		t.Fatalf("request path = %s, want %s (%s)", u.Path, path, rawURL)
	}

	return u.Query()
}
