package bybit

import (
	"strings"

	"github.com/plusev-terminal/go-plugin-common/logging"
	"github.com/plusev-terminal/go-plugin-common/plugin"
	rt "github.com/plusev-terminal/go-plugin-common/requester/types"
)


// Client represents a client for the Bybit V5 API
type Client struct {
	name        string
	baseURL     string
	wsBaseURL   string
	requester   rt.RequestDoer
	log         *logging.Logger
	apiKey      string // API key (decrypted)
	secretKey   string // API secret (decrypted)
	recvWindow  string // Request validity window in ms
	accountType string // "UNIFIED", "CONTRACT", "SPOT", "FUND"
	isTestnet   bool
}


// NewClient creates a new Bybit API client
func NewClient(req rt.RequestDoer, baseURL string, isTestnet bool) *Client {
	wsBaseURL := "wss://stream.bybit.com"
	if isTestnet {
		wsBaseURL = "wss://stream-testnet.bybit.com"
	}

	return &Client{
		name:        "Bybit",
		baseURL:     baseURL,
		wsBaseURL:   wsBaseURL,
		requester:   req,
		log:         logging.NewLogger("bybit-datasource"),
		recvWindow:  "5000",
		accountType: "UNIFIED",
		isTestnet:   isTestnet,
	}
}


func (c *Client) SetCredentials(creds map[string]string) {
	if apiKey, ok := creds["apiKey"]; ok {
		c.apiKey = strings.TrimSpace(apiKey)
	}

	if secretKey, ok := creds["secretKey"]; ok {
		c.secretKey = strings.TrimSpace(secretKey)
	}

	if at, ok := creds["accountType"]; ok {
		at = strings.TrimSpace(strings.ToUpper(at))
		if at != "" {
			c.accountType = at
		}
	}
}


// GetName returns the name of the data source
func (c *Client) GetName() string {
	return c.name
}


func (c *Client) GetConfigFields() []plugin.ConfigField {
	return []plugin.ConfigField{
		{
			Label:       "API Key",
			Name:        "apiKey",
			Description: "Your Bybit API key",
			Required:    true,
			Encrypt:     true,
			Mask:        true,
		},
		{
			Label:       "API Secret",
			Name:        "secretKey",
			Description: "Your Bybit API secret",
			Required:    true,
			Encrypt:     true,
			Mask:        true,
		},
		{
			Label:       "Account Type",
			Name:        "accountType",
			Description: "Bybit account type. Defaults to UNIFIED.",
			Required:    false,
			Type:        "select",
			DefaultOption: "UNIFIED",
			Options: []plugin.ConfigFieldSelectOption{
				{Value: "UNIFIED", Label: "Unified (UTA)"},
				{Value: "CONTRACT", Label: "Contract"},
				{Value: "SPOT", Label: "Spot"},
				{Value: "FUND", Label: "Funding"},
			},
		},
		{
			Label:       "Use Testnet",
			Name:        "testnet",
			Description: "Connect to Bybit testnet instead of mainnet",
			Required:    false,
			Type:        "bool",
		},
	}
}
