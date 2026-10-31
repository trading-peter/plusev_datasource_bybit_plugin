package bybit

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	rt "github.com/plusev-terminal/go-plugin-common/v2/requester/types"
	utils "github.com/plusev-terminal/go-plugin-common/v2/wasmutils"
)


// isAuthenticated returns true if the client has authentication credentials
func (c *Client) isAuthenticated() bool {
	return c.apiKey != "" && c.secretKey != ""
}


// addAuthHeaders adds Bybit V5 authentication headers to a request.
//
// Bybit signing: HMAC-SHA256 with the API secret as the key.
// Payload = timestamp + apiKey + recvWindow + queryString (GET) or jsonBody (POST).
// The signature is output as a lowercase hex string.
func (c *Client) addAuthHeaders(req *rt.Request, body string) {
	if !c.isAuthenticated() {
		return
	}

	now, _ := utils.Now()
	timestamp := now.UnixMilli()

	recvWindow := c.recvWindow
	if recvWindow == "" {
		recvWindow = "5000"
	}

	// Build the signing payload
	// GET: use the raw query string from the URL
	// POST: use the raw JSON body
	var paramStr string
	if req.Method == "GET" {
		if idx := strings.Index(req.URL, "?"); idx >= 0 {
			paramStr = req.URL[idx+1:]
		}
	} else {
		paramStr = body
	}

	payload := fmt.Sprintf("%d%s%s%s", timestamp, c.apiKey, recvWindow, paramStr)

	mac := hmac.New(sha256.New, []byte(c.secretKey))
	mac.Write([]byte(payload))
	signature := hex.EncodeToString(mac.Sum(nil))

	if req.Headers == nil {
		req.Headers = make(map[string]string)
	}

	req.Headers["X-BAPI-API-KEY"] = c.apiKey
	req.Headers["X-BAPI-TIMESTAMP"] = strconv.FormatInt(timestamp, 10)
	req.Headers["X-BAPI-SIGN"] = signature
	req.Headers["X-BAPI-RECV-WINDOW"] = recvWindow
}
