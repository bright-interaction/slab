package mollie

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewClient_StoresAPIKeyAndDefaultBase(t *testing.T) {
	c := NewClient("test_abc123")
	if c.apiKey != "test_abc123" {
		t.Errorf("apiKey: got %q", c.apiKey)
	}
	if c.baseURL != APIBase {
		t.Errorf("default baseURL: got %q, want %q", c.baseURL, APIBase)
	}
}

func TestClient_WithBaseURLChainable(t *testing.T) {
	c := NewClient("k").WithBaseURL("https://stub.local/v2")
	if c.baseURL != "https://stub.local/v2" {
		t.Errorf("baseURL: got %q", c.baseURL)
	}
}

func TestClient_IsConfigured(t *testing.T) {
	if NewClient("").IsConfigured() {
		t.Error("empty key should not be configured")
	}
	if !NewClient("test_xyz").IsConfigured() {
		t.Error("non-empty key should be configured")
	}
}

func TestClient_ModeFromKeyPrefix(t *testing.T) {
	cases := []struct {
		key  string
		want string
	}{
		{"test_abc", "test"},
		{"live_abc", "live"},
		{"abc", ""},
		{"", ""},
		{"TEST_abc", ""}, // case-sensitive
	}
	for _, c := range cases {
		if got := NewClient(c.key).Mode(); got != c.want {
			t.Errorf("Mode(%q) = %q, want %q", c.key, got, c.want)
		}
	}
}

func TestFormatAmount(t *testing.T) {
	cases := []struct {
		cents int64
		want  string
	}{
		{0, "0.00"},
		{1, "0.01"},
		{99, "0.99"},
		{100, "1.00"},
		{1234, "12.34"},
		{1999, "19.99"},
		{2995, "29.95"},
		{100000, "1000.00"},
		{-100, "-1.00"},
		{-50, "-0.50"},
	}
	for _, c := range cases {
		if got := FormatAmount(c.cents); got != c.want {
			t.Errorf("FormatAmount(%d) = %q, want %q", c.cents, got, c.want)
		}
	}
}

func TestCreatePayment_RequiresAPIKey(t *testing.T) {
	c := NewClient("")
	_, err := c.CreatePayment(context.Background(), CreatePaymentInput{})
	if err == nil {
		t.Error("CreatePayment with empty API key must error")
	}
	if !strings.Contains(err.Error(), "not configured") {
		t.Errorf("expected 'not configured' in error; got %q", err.Error())
	}
}

func TestGetPayment_RequiresAPIKey(t *testing.T) {
	c := NewClient("")
	_, err := c.GetPayment(context.Background(), "pay_x")
	if err == nil {
		t.Error("GetPayment with empty API key must error")
	}
}

func TestCreateRefund_RequiresAPIKey(t *testing.T) {
	c := NewClient("")
	_, err := c.CreateRefund(context.Background(), "pay_x", CreateRefundInput{})
	if err == nil {
		t.Error("CreateRefund with empty API key must error")
	}
}

func TestCreatePayment_SendsCorrectAuthHeaderAndBody(t *testing.T) {
	var gotAuth, gotBody, gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotMethod = r.Method
		gotPath = r.URL.Path
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"id":"tr_abc","status":"open","amount":{"currency":"EUR","value":"29.95"},"_links":{"checkout":{"href":"https://www.mollie.com/checkout/x","type":"text/html"}}}`)
	}))
	defer srv.Close()
	c := NewClient("test_key").WithBaseURL(srv.URL)
	p, err := c.CreatePayment(context.Background(), CreatePaymentInput{
		Amount:      MoneyValue{Currency: "EUR", Value: "29.95"},
		Description: "order-123",
		RedirectURL: "https://shop.example.com/done",
	})
	if err != nil {
		t.Fatalf("CreatePayment: %v", err)
	}
	if gotMethod != http.MethodPost {
		t.Errorf("method: got %s, want POST", gotMethod)
	}
	if gotPath != "/payments" {
		t.Errorf("path: got %s, want /payments", gotPath)
	}
	if gotAuth != "Bearer test_key" {
		t.Errorf("auth header: got %q", gotAuth)
	}
	if !strings.Contains(gotBody, `"currency":"EUR"`) {
		t.Errorf("body missing amount: %s", gotBody)
	}
	if p.ID != "tr_abc" {
		t.Errorf("payment id: got %q", p.ID)
	}
	if p.Links.Checkout == nil || p.Links.Checkout.Href == "" {
		t.Errorf("checkout link missing: %+v", p.Links)
	}
}

func TestGetPayment_BuildsCorrectPath(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_ = json.NewEncoder(w).Encode(Payment{ID: "tr_xyz", Status: "paid"})
	}))
	defer srv.Close()
	c := NewClient("test_key").WithBaseURL(srv.URL)
	p, err := c.GetPayment(context.Background(), "tr_xyz")
	if err != nil {
		t.Fatalf("GetPayment: %v", err)
	}
	if gotPath != "/payments/tr_xyz" {
		t.Errorf("path: got %s", gotPath)
	}
	if p.Status != "paid" {
		t.Errorf("status: got %s", p.Status)
	}
}

func TestPropagatesUpstreamErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(422)
		_, _ = io.WriteString(w, `{"status":422,"title":"Unprocessable Entity"}`)
	}))
	defer srv.Close()
	c := NewClient("test_key").WithBaseURL(srv.URL)
	_, err := c.CreatePayment(context.Background(), CreatePaymentInput{})
	if err == nil {
		t.Fatal("expected error for 422 response")
	}
	if !strings.Contains(err.Error(), "422") {
		t.Errorf("error should include status; got %q", err.Error())
	}
	if !strings.Contains(err.Error(), "Unprocessable") {
		t.Errorf("error should include body; got %q", err.Error())
	}
}

func TestCreateRefund_PathAndBody(t *testing.T) {
	var gotPath, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		_, _ = io.WriteString(w, `{"id":"re_1","amount":{"currency":"EUR","value":"5.00"},"status":"pending","paymentId":"tr_abc","createdAt":"2026-06-01T00:00:00Z"}`)
	}))
	defer srv.Close()
	c := NewClient("test_key").WithBaseURL(srv.URL)
	r, err := c.CreateRefund(context.Background(), "tr_abc", CreateRefundInput{
		Amount: MoneyValue{Currency: "EUR", Value: "5.00"}, Description: "partial",
	})
	if err != nil {
		t.Fatalf("CreateRefund: %v", err)
	}
	if gotPath != "/payments/tr_abc/refunds" {
		t.Errorf("path: got %s", gotPath)
	}
	if !strings.Contains(gotBody, `"value":"5.00"`) {
		t.Errorf("body missing amount: %s", gotBody)
	}
	if r.ID != "re_1" || r.PaymentID != "tr_abc" {
		t.Errorf("refund: %+v", r)
	}
}
