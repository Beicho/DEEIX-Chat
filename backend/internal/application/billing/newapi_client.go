package billing

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	newAPIQuotaPerUSD         = 500000.0
	newAPITransferRate        = 10.0
	newAPIMinTransferUSD      = 1.0
	newAPIMaxDailyTransferUSD = 500.0
)

// NewAPIClientConfig configures the NewAPI bridge client.
type NewAPIClientConfig struct {
	BaseURL string
	HMACKey string
	Timeout time.Duration
}

// NewAPIClient calls NewAPI's HMAC-protected bridge API.
type NewAPIClient struct {
	baseURL    string
	hmacKey    string
	httpClient *http.Client
}

// NewAPIBridgeUser describes a NewAPI user returned by the bridge.
type NewAPIBridgeUser struct {
	ExternalUserID string
	Username       string
	BalanceUSD     float64
}

// NewAPITransferOutResult describes a reserved transfer on NewAPI.
type NewAPITransferOutResult struct {
	TransferID string
}

// NewNewAPIClient creates a bridge client.
func NewNewAPIClient(cfg NewAPIClientConfig) *NewAPIClient {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	return &NewAPIClient{
		baseURL: strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/"),
		hmacKey: strings.TrimSpace(cfg.HMACKey),
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

func (c *NewAPIClient) Configured() bool {
	return c != nil && c.baseURL != "" && c.hmacKey != ""
}

func (c *NewAPIClient) UserByLinuxDOSub(ctx context.Context, sub string) (*NewAPIBridgeUser, error) {
	values := url.Values{}
	values.Set("sub", strings.TrimSpace(sub))
	var payload struct {
		UserID   interface{} `json:"user_id"`
		Username string      `json:"username"`
		Quota    int64       `json:"quota"`
	}
	if err := c.doJSON(ctx, http.MethodGet, "/api/bridge/user-by-linuxdo?"+values.Encode(), nil, &payload); err != nil {
		return nil, err
	}
	externalUserID := stringifyNewAPIUserID(payload.UserID)
	if externalUserID == "" {
		return nil, fmt.Errorf("newapi bridge response missing user")
	}
	return &NewAPIBridgeUser{
		ExternalUserID: externalUserID,
		Username:       strings.TrimSpace(payload.Username),
		BalanceUSD:     quotaToUSD(payload.Quota),
	}, nil
}

func (c *NewAPIClient) TransferOut(ctx context.Context, externalUserID string, amountUSD float64, idempotencyKey string) (*NewAPITransferOutResult, error) {
	quotaAmount := usdToNewAPIQuota(amountUSD)
	if strings.TrimSpace(externalUserID) == "" || quotaAmount <= 0 || strings.TrimSpace(idempotencyKey) == "" {
		return nil, fmt.Errorf("invalid newapi transfer input")
	}
	body := map[string]interface{}{
		"user_id":         externalUserID,
		"quota_amount":    quotaAmount,
		"idempotency_key": strings.TrimSpace(idempotencyKey),
	}
	var payload struct {
		TransferID interface{} `json:"transfer_id"`
	}
	if err := c.doJSON(ctx, http.MethodPost, "/api/bridge/transfer-out", body, &payload); err != nil {
		return nil, err
	}
	transferID := stringifyNewAPIUserID(payload.TransferID)
	if transferID == "" {
		return nil, fmt.Errorf("newapi bridge response missing transfer")
	}
	return &NewAPITransferOutResult{TransferID: transferID}, nil
}

func (c *NewAPIClient) TransferConfirm(ctx context.Context, transferID string, idempotencyKey string) error {
	return c.postTransferState(ctx, "/api/bridge/transfer-confirm", transferID, idempotencyKey)
}

func (c *NewAPIClient) TransferCancel(ctx context.Context, transferID string, idempotencyKey string) error {
	return c.postTransferState(ctx, "/api/bridge/transfer-cancel", transferID, idempotencyKey)
}

func (c *NewAPIClient) postTransferState(ctx context.Context, path string, transferID string, idempotencyKey string) error {
	body := map[string]interface{}{
		"transfer_id":     strings.TrimSpace(transferID),
		"idempotency_key": strings.TrimSpace(idempotencyKey),
	}
	return c.doJSON(ctx, http.MethodPost, path, body, nil)
}

func (c *NewAPIClient) doJSON(ctx context.Context, method string, path string, body interface{}, out interface{}) error {
	if !c.Configured() {
		return fmt.Errorf("newapi bridge is unavailable")
	}
	var raw []byte
	var err error
	if body != nil {
		raw, err = json.Marshal(body)
		if err != nil {
			return err
		}
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	c.sign(req, raw)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("newapi bridge returned %s", resp.Status)
	}
	if out == nil || len(respBody) == 0 {
		return nil
	}
	if err := json.Unmarshal(respBody, out); err != nil {
		var wrapped struct {
			Data json.RawMessage `json:"data"`
		}
		if wrapErr := json.Unmarshal(respBody, &wrapped); wrapErr == nil && len(wrapped.Data) > 0 {
			return json.Unmarshal(wrapped.Data, out)
		}
		return err
	}
	return nil
}

func (c *NewAPIClient) sign(req *http.Request, body []byte) {
	timestamp := strconv.FormatInt(time.Now().UTC().Unix(), 10)
	nonce := randomHex(12)
	req.Header.Set("X-DEEIX-Timestamp", timestamp)
	req.Header.Set("X-DEEIX-Nonce", nonce)
	mac := hmac.New(sha256.New, []byte(c.hmacKey))
	_, _ = mac.Write([]byte(req.Method))
	_, _ = mac.Write([]byte("\n"))
	_, _ = mac.Write([]byte(req.URL.RequestURI()))
	_, _ = mac.Write([]byte("\n"))
	_, _ = mac.Write([]byte(timestamp))
	_, _ = mac.Write([]byte("\n"))
	_, _ = mac.Write([]byte(nonce))
	_, _ = mac.Write([]byte("\n"))
	_, _ = mac.Write(body)
	req.Header.Set("X-DEEIX-Signature", "sha256="+hex.EncodeToString(mac.Sum(nil)))
}

func randomHex(size int) string {
	if size <= 0 {
		size = 12
	}
	raw := make([]byte, size)
	if _, err := rand.Read(raw); err != nil {
		return strconv.FormatInt(time.Now().UTC().UnixNano(), 16)
	}
	return hex.EncodeToString(raw)
}

func quotaToUSD(quota int64) float64 {
	if quota <= 0 {
		return 0
	}
	return float64(quota) / newAPIQuotaPerUSD
}

func usdToNewAPIQuota(value float64) int64 {
	if !isPositiveFinite(value) {
		return 0
	}
	return int64(math.Round(value * newAPIQuotaPerUSD))
}

func stringifyNewAPIUserID(value interface{}) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case float64:
		if typed <= 0 {
			return ""
		}
		return strconv.FormatInt(int64(typed), 10)
	case int:
		if typed <= 0 {
			return ""
		}
		return strconv.Itoa(typed)
	case int64:
		if typed <= 0 {
			return ""
		}
		return strconv.FormatInt(typed, 10)
	case json.Number:
		return strings.TrimSpace(typed.String())
	default:
		return ""
	}
}
