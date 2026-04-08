package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
)

var (
	db  *sql.DB
	rdb *redis.Client
	ctx = context.Background()
)

func main() {

	//DB Connection
	var err error
	connStr := "postgres://user:password@localhost:5434/ticket_db?sslmode=disable"
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		panic(err)
	}

	// Redis connection
	rdb = redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	// CRITICAL: sync stock quantity from DB to redis
	var stock int
	err = db.QueryRow("SELECT stock FROM tickets WHERE id = 1").Scan(&stock)
	if err != nil {
		panic(err)
	}
	rdb.Set(ctx, "ticket_stock:1", stock, 0)

	r := chi.NewRouter()
	r.Post("/buy", buyWithRedisShield)

	fmt.Println("Redis shield active! Server runs at port: 8080")
	http.ListenAndServe(":8080", r)
}

func buyWithRedisShield(w http.ResponseWriter, r *http.Request) {
	//Step A: decrease stock from redis
	newStock, err := rdb.Decr(ctx, "ticket_stock:1").Result()
	if err != nil {
		http.Error(w, "Redis Error", http.StatusInternalServerError)
		return
	}

	//Step B: if new stock is lower than 0 tickets sold out
	if newStock < 0 {
		http.Error(w, "Tickets sold out!", http.StatusConflict)
		return
	}

	//Step C: Redis approved, update DB quietly
	_, err = db.Exec("UPDATE tickets SET stock = stock - 1 WHERE id = 1")
	if err != nil {
		fmt.Println("DB could not updated: ", err)
	}

	w.Write([]byte("Your ticket purchased succesfully (redis approved)"))
}
