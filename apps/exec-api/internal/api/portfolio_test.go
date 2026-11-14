package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	calcv1 "proto/gen"
)

func decodePortfolio(t *testing.T, body []byte) PortfolioResponse {
	t.Helper()
	var resp PortfolioResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("decoding body: %v (%s)", err, body)
	}
	return resp
}

// User 1 has A:40 B:60 in the shipped data/portfolio.yml.
func TestGetPortfolioHandler_Existing(t *testing.T) {
	server := testServer(t)
	w := doRequestAs(t, server, 1, http.MethodGet, "/api/portfolio", "")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", w.Code, w.Body)
	}
	got := decodePortfolio(t, w.Body.Bytes()).TargetPortfolio
	if got["A"] != 40 || got["B"] != 60 || len(got) != 2 {
		t.Errorf("target_portfolio = %v, want A:40 B:60", got)
	}
}

func TestGetPortfolioHandler_NotFound(t *testing.T) {
	server := testServer(t)
	w := doRequestAs(t, server, 500, http.MethodGet, "/api/portfolio", "")
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404; body = %s", w.Code, w.Body)
	}
}

// A new user sets a portfolio, reads it back, and can then trade with it.
func TestSetPortfolioHandler_CreateThenReadThenReplace(t *testing.T) {
	server := testServer(t)

	w := doRequestAs(t, server, 500, http.MethodPut, "/api/portfolio", `{"target_portfolio":{"A":30,"B":70}}`)
	if w.Code != http.StatusOK {
		t.Fatalf("PUT status = %d, want 200; body = %s", w.Code, w.Body)
	}
	if got := decodePortfolio(t, w.Body.Bytes()).TargetPortfolio; got["A"] != 30 || got["B"] != 70 {
		t.Errorf("PUT response = %v, want A:30 B:70", got)
	}

	w = doRequestAs(t, server, 500, http.MethodGet, "/api/portfolio", "")
	if w.Code != http.StatusOK {
		t.Fatalf("GET status = %d, want 200", w.Code)
	}
	if got := decodePortfolio(t, w.Body.Bytes()).TargetPortfolio; got["A"] != 30 || got["B"] != 70 {
		t.Errorf("GET after PUT = %v, want A:30 B:70", got)
	}

	w = doRequestAs(t, server, 500, http.MethodPut, "/api/portfolio", `{"target_portfolio":{"C":100}}`)
	if w.Code != http.StatusOK {
		t.Fatalf("second PUT status = %d, want 200", w.Code)
	}
	w = doRequestAs(t, server, 500, http.MethodGet, "/api/portfolio", "")
	if got := decodePortfolio(t, w.Body.Bytes()).TargetPortfolio; len(got) != 1 || got["C"] != 100 {
		t.Errorf("GET after replace = %v, want only C:100", got)
	}
}

// Setting a portfolio is per user: it must not touch anyone else's.
func TestSetPortfolioHandler_OnlyAffectsTokenOwner(t *testing.T) {
	server := testServer(t)

	if w := doRequestAs(t, server, 500, http.MethodPut, "/api/portfolio", `{"target_portfolio":{"C":100}}`); w.Code != http.StatusOK {
		t.Fatalf("PUT status = %d", w.Code)
	}
	w := doRequestAs(t, server, 1, http.MethodGet, "/api/portfolio", "")
	if got := decodePortfolio(t, w.Body.Bytes()).TargetPortfolio; got["A"] != 40 || got["B"] != 60 {
		t.Errorf("user 1 portfolio = %v, want unchanged A:40 B:60", got)
	}
}

func TestSetPortfolioHandler_Invalid(t *testing.T) {
	cases := map[string]struct {
		body    string
		errPart string
	}{
		"sum not 100":    {`{"target_portfolio":{"A":40,"B":50}}`, "not 100%"},
		"negative":       {`{"target_portfolio":{"A":150,"B":-50}}`, "between 1 and 100"},
		"zero weight":    {`{"target_portfolio":{"A":100,"B":0}}`, "between 1 and 100"},
		"unknown ticker": {`{"target_portfolio":{"A":50,"ZZZ":50}}`, "ZZZ"},
		"blank ticker":   {`{"target_portfolio":{" ":100}}`, "symbol"},
		"empty":          {`{"target_portfolio":{}}`, ""},
		"missing field":  {`{}`, ""},
		"not a number":   {`{"target_portfolio":{"A":"x"}}`, ""},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			server := testServer(t)
			w := doRequestAs(t, server, 500, http.MethodPut, "/api/portfolio", c.body)
			if w.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400; body = %s", w.Code, w.Body)
			}
			if !strings.Contains(w.Body.String(), c.errPart) {
				t.Errorf("body = %s, want it to mention %q", w.Body, c.errPart)
			}
			// Nothing was stored.
			if w := doRequestAs(t, server, 500, http.MethodGet, "/api/portfolio", ""); w.Code != http.StatusNotFound {
				t.Errorf("GET after invalid PUT status = %d, want 404", w.Code)
			}
		})
	}
}

func TestPortfolioRoutes_RequireAuth(t *testing.T) {
	server := testServer(t)
	for _, method := range []string{http.MethodGet, http.MethodPut} {
		w := doRequest(server, method, "/api/portfolio", `{"target_portfolio":{"A":100}}`)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("%s status = %d, want 401", method, w.Code)
		}
	}
}

// The point of the endpoint: a user with no portfolio can't trade (404), and
// after setting one they can.
func TestSetPortfolioHandler_EnablesTrading(t *testing.T) {
	calc := &fakeCalcClient{resp: &calcv1.CalculateResponse{Orders: []*calcv1.Order{
		{Symbol: "A", Amount: 10000, QuantityUnits: 10000},
	}}}
	server := testServerWithCalc(t, calc)

	if w := doRequestAs(t, server, 500, http.MethodPost, "/api/trades", `{"amount": 10000}`); w.Code != http.StatusNotFound {
		t.Fatalf("trade before portfolio: status = %d, want 404", w.Code)
	}
	if w := doRequestAs(t, server, 500, http.MethodPut, "/api/portfolio", `{"target_portfolio":{"A":100}}`); w.Code != http.StatusOK {
		t.Fatalf("PUT portfolio: status = %d, want 200; body = %s", w.Code, w.Body)
	}
	w := doRequestAs(t, server, 500, http.MethodPost, "/api/trades", `{"amount": 10000}`)
	if w.Code != http.StatusOK {
		t.Fatalf("trade after portfolio: status = %d, want 200; body = %s", w.Code, w.Body)
	}
	var trade TradeResponse
	if err := json.Unmarshal(w.Body.Bytes(), &trade); err != nil {
		t.Fatal(err)
	}
	if trade.TargetPortfolio["A"] != 100 {
		t.Errorf("trade target_portfolio = %v, want A:100", trade.TargetPortfolio)
	}
}
