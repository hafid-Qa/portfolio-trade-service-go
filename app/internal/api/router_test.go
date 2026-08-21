package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"app/config"

	"github.com/gin-gonic/gin"
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	os.Exit(m.Run())
}

// testServer builds a real Server against the shipped reference data
// (data/stocks.yml, data/portfolio.yml), the same files the app runs against
// in production -- this is what proves the shipped configuration produces the
// documented numbers, not a hand-rolled fixture that could drift from it.
func testServer(t *testing.T) *Server {
	t.Helper()
	cfg := &config.Config{
		StockPath:     "../../../data/stocks.yml",
		PortfolioPath: "../../../data/portfolio.yml",
	}
	server, err := NewServer(cfg)
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}
	return server
}

func doRequest(server *Server, method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	server.router.ServeHTTP(w, req)
	return w
}

func TestHealthHandler(t *testing.T) {
	server := testServer(t)
	w := doRequest(server, http.MethodGet, "/api/health", "")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body = %s", w.Code, w.Body.String())
	}
}

// TestTradeHandler_HappyPath is user 1 from the shipped data (A:40, B:60),
// matching the documented spec example exactly.
func TestTradeHandler_HappyPath(t *testing.T) {
	server := testServer(t)
	w := doRequest(server, http.MethodPost, "/api/users/1/trade", `{"amount": 10000}`)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body = %s", w.Code, w.Body.String())
	}

	var resp TradeResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v (body=%s)", err, w.Body.String())
	}
	if resp.Amount != 10000 {
		t.Errorf("Amount = %d, want 10000", resp.Amount)
	}
	if len(resp.Orders) != 2 {
		t.Fatalf("len(Orders) = %d, want 2: %+v", len(resp.Orders), resp.Orders)
	}
	want := map[string]OrderResponse{
		"A": {Symbol: "A", Amount: 4000, Quantity: 4.0},
		"B": {Symbol: "B", Amount: 6000, Quantity: 38.709},
	}
	for _, got := range resp.Orders {
		w, ok := want[got.Symbol]
		if !ok || got != w {
			t.Errorf("order %+v, want %+v (ok=%v)", got, w, ok)
		}
	}
}

func TestTradeHandler_UnknownUser(t *testing.T) {
	server := testServer(t)
	w := doRequest(server, http.MethodPost, "/api/users/999/trade", `{"amount": 10000}`)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404, body = %s", w.Code, w.Body.String())
	}
}

func TestTradeHandler_AmountBelowMinimum(t *testing.T) {
	server := testServer(t)
	w := doRequest(server, http.MethodPost, "/api/users/1/trade", `{"amount": 500}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400, body = %s", w.Code, w.Body.String())
	}
}

func TestTradeHandler_InvalidUserID(t *testing.T) {
	server := testServer(t)
	w := doRequest(server, http.MethodPost, "/api/users/0/trade", `{"amount": 10000}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400, body = %s", w.Code, w.Body.String())
	}
}

// TestTradeHandler_HaltedOnlyPortfolio is user 2 (E:100, halted): a fully
// halted portfolio must return an empty order list with 200, not an error.
func TestTradeHandler_HaltedOnlyPortfolio(t *testing.T) {
	server := testServer(t)
	w := doRequest(server, http.MethodPost, "/api/users/2/trade", `{"amount": 10000}`)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body = %s", w.Code, w.Body.String())
	}

	var resp TradeResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(resp.Orders) != 0 {
		t.Errorf("len(Orders) = %d, want 0", len(resp.Orders))
	}
}

// TestNewServer_RejectsPortfolioReferencingUnknownTicker is the startup
// referential-integrity check: a portfolio referencing a ticker absent from
// the stock catalogue must fail NewServer, not surface as a 500 on a request.
func TestNewServer_RejectsPortfolioReferencingUnknownTicker(t *testing.T) {
	dir := t.TempDir()
	stocksPath := filepath.Join(dir, "stocks.yml")
	portfolioPath := filepath.Join(dir, "portfolio.yml")

	if err := os.WriteFile(stocksPath, []byte(`
- ticker: "A"
  price: 1000
  tradable: true
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(portfolioPath, []byte(`
- user_id: 1
  target_portfolio:
    A: 40
    ZZZ: 60
`), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{StockPath: stocksPath, PortfolioPath: portfolioPath}
	_, err := NewServer(cfg)
	if err == nil {
		t.Fatal("NewServer() error = nil, want an error for a portfolio referencing an unknown ticker")
	}
}
