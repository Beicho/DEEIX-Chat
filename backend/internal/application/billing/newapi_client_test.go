package billing

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewAPIClientSignsBridgeRequests(t *testing.T) {
	const secret = "bridge-secret"
	var seenPath string
	var seenSignature string
	var seenTimestamp string
	var seenNonce string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		seenPath = r.URL.RequestURI()
		seenSignature = r.Header.Get("X-DEEIX-Signature")
		seenTimestamp = r.Header.Get("X-DEEIX-Timestamp")
		seenNonce = r.Header.Get("X-DEEIX-Nonce")
		if seenTimestamp == "" || seenNonce == "" {
			t.Fatalf("missing hmac timestamp or nonce")
		}
		mac := hmac.New(sha256.New, []byte(secret))
		_, _ = mac.Write([]byte(r.Method))
		_, _ = mac.Write([]byte("\n"))
		_, _ = mac.Write([]byte(r.URL.RequestURI()))
		_, _ = mac.Write([]byte("\n"))
		_, _ = mac.Write([]byte(seenTimestamp))
		_, _ = mac.Write([]byte("\n"))
		_, _ = mac.Write([]byte(seenNonce))
		_, _ = mac.Write([]byte("\n"))
		_, _ = mac.Write(body)
		expected := "sha256=" + hex.EncodeToString(mac.Sum(nil))
		if !hmac.Equal([]byte(expected), []byte(seenSignature)) {
			t.Fatalf("unexpected signature: got %q want %q", seenSignature, expected)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"user_id":1001,"username":"alice","quota":5000000}`))
	}))
	defer server.Close()

	client := NewNewAPIClient(NewAPIClientConfig{
		BaseURL: server.URL,
		HMACKey: secret,
	})
	user, err := client.UserByLinuxDOSub(context.Background(), "linuxdo-sub")
	if err != nil {
		t.Fatalf("user by linuxdo sub: %v", err)
	}
	if user.ExternalUserID != "1001" || user.BalanceUSD != 10 {
		t.Fatalf("unexpected user response: %#v", user)
	}
	if !strings.HasPrefix(seenPath, "/api/bridge/user-by-linuxdo?") {
		t.Fatalf("unexpected signed path: %s", seenPath)
	}
	if seenSignature == "" {
		t.Fatal("expected signature header")
	}
}
