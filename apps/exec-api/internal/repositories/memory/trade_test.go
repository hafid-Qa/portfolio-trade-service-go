package memory

import (
	"context"
	"errors"
	"testing"

	"app/internal/domain"
)

func TestTradeRepo_CreateAndGet(t *testing.T) {
	repo := NewTradeRepo()
	orders := []domain.Order{domain.NewOrder("A", 4000, 4000)}

	created, err := repo.Create(context.Background(), 1, 10000, map[domain.Symbol]int{"A": 100}, orders, domain.TradeStatusPending)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.ID() == 0 {
		t.Error("Create() ID() = 0, want a real assigned ID")
	}

	got, err := repo.Get(context.Background(), created.ID())
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.ID() != created.ID() || got.UserID() != 1 || got.Amount() != 10000 {
		t.Errorf("Get() = %+v, want a match for the created trade", got)
	}
}

func TestTradeRepo_Get_NotFound(t *testing.T) {
	repo := NewTradeRepo()
	_, err := repo.Get(context.Background(), 999)
	if !errors.Is(err, domain.ErrTradeNotFound) {
		t.Errorf("Get() error = %v, want ErrTradeNotFound", err)
	}
}

func TestTradeRepo_GetByUserID(t *testing.T) {
	repo := NewTradeRepo()
	repo.Create(context.Background(), 1, 10000, map[domain.Symbol]int{"A": 100}, nil, domain.TradeStatusPending)
	repo.Create(context.Background(), 1, 5000, map[domain.Symbol]int{"B": 100}, nil, domain.TradeStatusCompleted)
	repo.Create(context.Background(), 2, 1000, map[domain.Symbol]int{"A": 100}, nil, domain.TradeStatusPending)

	got, err := repo.GetByUserID(context.Background(), 1)
	if err != nil {
		t.Fatalf("GetByUserID() error = %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("GetByUserID(1) returned %d trades, want 2", len(got))
	}
	for _, trade := range got {
		if trade.UserID() != 1 {
			t.Errorf("GetByUserID(1) returned a trade for user %d", trade.UserID())
		}
	}
}

func TestTradeRepo_GetByUserID_NoTrades(t *testing.T) {
	repo := NewTradeRepo()
	got, err := repo.GetByUserID(context.Background(), 999)
	if err != nil {
		t.Fatalf("GetByUserID() error = %v", err)
	}
	if len(got) != 0 {
		t.Errorf("GetByUserID() returned %d trades, want 0", len(got))
	}
}
