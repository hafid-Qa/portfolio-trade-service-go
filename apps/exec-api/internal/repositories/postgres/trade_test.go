package postgres

import (
	"context"
	"errors"
	"testing"

	"app/internal/domain"
)

func seedTrade(t *testing.T, userID int64, amount int, targetPortfolio map[domain.Symbol]int, orders []domain.Order, status domain.TradeStatus) domain.TradeResult {
	t.Helper()
	seedUser(t, userID)
	repo := NewTradeRepo(testStore)
	trade, err := repo.Create(context.Background(), userID, amount, targetPortfolio, orders, status)
	if err != nil {
		t.Fatalf("seeding trade for user %d: %v", userID, err)
	}
	t.Cleanup(func() {
		testStore.DeleteTrade(context.Background(), int32(trade.ID()))
	})
	return trade
}

func TestTradeRepo_Create(t *testing.T) {
	orders := []domain.Order{domain.NewOrder("A", 4000, 4000)}
	trade := seedTrade(t, 90101, 10000, map[domain.Symbol]int{"A": 100}, orders, domain.TradeStatusPending)

	if trade.ID() == 0 {
		t.Error("Create() ID() = 0, want a real assigned ID")
	}
	if trade.UserID() != 90101 {
		t.Errorf("UserID() = %d, want 90101", trade.UserID())
	}
	if trade.Amount() != 10000 {
		t.Errorf("Amount() = %d, want 10000", trade.Amount())
	}
	if trade.Status() != domain.TradeStatusPending {
		t.Errorf("Status() = %v, want pending", trade.Status())
	}
	if trade.CreatedAt().IsZero() {
		t.Error("CreatedAt() is zero, want a real timestamp")
	}
}

func TestTradeRepo_Get(t *testing.T) {
	orders := []domain.Order{domain.NewOrder("A", 4000, 4000), domain.NewOrder("B", 6000, 38709)}
	seeded := seedTrade(t, 90102, 10000, map[domain.Symbol]int{"A": 40, "B": 60}, orders, domain.TradeStatusPending)

	repo := NewTradeRepo(testStore)
	got, err := repo.Get(context.Background(), seeded.ID())
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.ID() != seeded.ID() {
		t.Errorf("Get().ID() = %d, want %d", got.ID(), seeded.ID())
	}
	target := got.TargetPortfolio()
	if target["A"] != 40 || target["B"] != 60 {
		t.Errorf("TargetPortfolio() = %v, want A:40 B:60", target)
	}
	if len(got.Orders()) != 2 {
		t.Fatalf("len(Orders()) = %d, want 2", len(got.Orders()))
	}
}

func TestTradeRepo_Get_NotFound(t *testing.T) {
	repo := NewTradeRepo(testStore)
	_, err := repo.Get(context.Background(), 999999)
	if !errors.Is(err, domain.ErrTradeNotFound) {
		t.Errorf("Get() error = %v, want ErrTradeNotFound", err)
	}
}

func TestTradeRepo_GetByUserID(t *testing.T) {
	seedTrade(t, 90103, 10000, map[domain.Symbol]int{"A": 100}, nil, domain.TradeStatusPending)
	seedTrade(t, 90103, 5000, map[domain.Symbol]int{"B": 100}, nil, domain.TradeStatusCompleted)
	seedTrade(t, 90104, 1000, map[domain.Symbol]int{"A": 100}, nil, domain.TradeStatusPending)

	repo := NewTradeRepo(testStore)
	got, err := repo.GetByUserID(context.Background(), 90103)
	if err != nil {
		t.Fatalf("GetByUserID() error = %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("GetByUserID() returned %d trades, want 2 (only user 90103's)", len(got))
	}
	for _, trade := range got {
		if trade.UserID() != 90103 {
			t.Errorf("GetByUserID(90103) returned a trade for user %d", trade.UserID())
		}
	}
}

func TestTradeRepo_GetByUserID_NoTrades(t *testing.T) {
	repo := NewTradeRepo(testStore)
	got, err := repo.GetByUserID(context.Background(), 999999)
	if err != nil {
		t.Fatalf("GetByUserID() error = %v", err)
	}
	if len(got) != 0 {
		t.Errorf("GetByUserID() returned %d trades, want 0", len(got))
	}
}
