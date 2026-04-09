package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/go-redsync/redsync/v4"
	goredis "github.com/go-redsync/redsync/v4/redis/goredis/v9"
	"github.com/redis/go-redis/v9"
)

type ticketRepo struct {
	db    *sql.DB
	rdb   *redis.Client
	rsync *redsync.Redsync
}

func NewTicketRepository(db *sql.DB, rdb *redis.Client) *ticketRepo {
	pool := goredis.NewPool(rdb)
	rsync := redsync.New(pool)
	return &ticketRepo{db: db, rdb: rdb, rsync: rsync}
}

func (r *ticketRepo) DecrementStockCache(ctx context.Context, id int) (int64, error) {
	key := fmt.Sprintf("ticket_stock:%d", id)
	return r.rdb.Decr(ctx, key).Result()
}

func (r *ticketRepo) UpdateStockDB(ctx context.Context, id int, quantity int) error {
	_, err := r.db.ExecContext(ctx, "UPDATE tickets SET stock = stock - $1 WHERE id = $2 AND stock >= $1", quantity, id)
	return err
}

func (r *ticketRepo) GetStock(ctx context.Context, id int) (int, error) {
	var stock int
	err := r.db.QueryRowContext(ctx, "SELECT stock FROM tickets WHERE id = $1", id).Scan(&stock)
	return stock, err
}

func (r *ticketRepo) SyncCacheWithDB(ctx context.Context, id int) error {
	stock, err := r.GetStock(ctx, id)
	if err != nil {
		return err
	}
	key := fmt.Sprintf("ticket_stock:%d", id)
	return r.rdb.Set(ctx, key, stock, 0).Err()
}

// Acquires lock for a specific resource (e.g. ticket id)
func (r *ticketRepo) AcquireLock(ctx context.Context, resourceID string) (*redsync.Mutex, error) {
	mutex := r.rsync.NewMutex("lock:"+resourceID,
		redsync.WithExpiry(5*time.Second),
		redsync.WithTries(3),
	)

	if err := mutex.LockContext(ctx); err != nil {
		return nil, err
	}
	return mutex, nil
}
