package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/redis/go-redis/v9"
)

type TicketRepo struct {
	db  *sql.DB
	rdb *redis.Client
}

func NewTicketRepository(db *sql.DB, rdb *redis.Client) *TicketRepo {
	return &TicketRepo{db: db, rdb: rdb}
}

func (r *TicketRepo) DecrementStockCache(ctx context.Context, id int) (int64, error) {
	key := fmt.Sprintf("ticket_stock:%d", id)
	return r.rdb.Decr(ctx, key).Result()
}

func (r *TicketRepo) UpdateStockDB(ctx context.Context, id int, quantity int) error {
	_, err := r.db.ExecContext(ctx, "UPDATE tickets SET stock = stock - $1 WHERE id = $2 AND stock >= $1", quantity, id)
	return err
}

func (r *TicketRepo) GetStock(ctx context.Context, id int) (int, error) {
	var stock int
	err := r.db.QueryRowContext(ctx, "SELECT stock FROM tickets WHERE id = $1", id).Scan(&stock)
	return stock, err
}

func (r *TicketRepo) SyncCacheWithDB(ctx context.Context, id int) error {
	stock, err := r.GetStock(ctx, id)
	if err != nil {
		return err
	}
	key := fmt.Sprintf("ticket_stock:%d", id)
	return r.rdb.Set(ctx, key, stock, 0).Err()
}
