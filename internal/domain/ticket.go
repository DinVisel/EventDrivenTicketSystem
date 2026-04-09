package domain

import (
	"context"

	"github.com/go-redsync/redsync/v4"
)

type Ticket struct {
	ID        int    `json:"id"`
	EventName string `json:"event_name"`
	Stock     int    `json:"stock"`
}

type TicketRepository interface {
	GetStock(ctx context.Context, id int) (int, error)
	UpdateStockDB(ctx context.Context, id int, quantity int) error
	DecrementStockCache(ctx context.Context, id int) (int64, error)
	SyncCacheWithDB(ctx context.Context, id int) error
	AcquireLock(ctx context.Context, resourceId string) (*redsync.Mutex, error)
}

type TicketService interface {
	Purchase(ctx context.Context, ticketID int) error
}
