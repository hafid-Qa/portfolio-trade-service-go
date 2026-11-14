package api

import (
	"encoding/json"
	"net/http"
	"testing"
)

// The shipped data/stocks.yml has A-E, with E halted.
func TestListStocksHandler(t *testing.T) {
	server := testServer(t)

	w := doRequestAs(t, server, 1, http.MethodGet, "/api/stocks", "")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", w.Code, w.Body)
	}

	var got []StockResponse
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decoding body: %v", err)
	}
	want := []StockResponse{
		{Symbol: "A", Price: 1000, Tradable: true},
		{Symbol: "B", Price: 155, Tradable: true},
		{Symbol: "C", Price: 2222, Tradable: true},
		{Symbol: "D", Price: 467, Tradable: true},
		{Symbol: "E", Price: 888, Tradable: false},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d stocks, want %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("stock %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestListStocksHandler_RequiresAuth(t *testing.T) {
	w := doRequest(testServer(t), http.MethodGet, "/api/stocks", "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}
