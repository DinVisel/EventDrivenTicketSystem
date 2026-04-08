package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	_ "github.com/lib/pq"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
)

var (
	db       *sql.DB
	rdb      *redis.Client
	amqpChan *amqp.Channel
	ctx      = context.Background()
)

func main() {
	// DB redis connection
	initDBAndRedis()

	// RabbitMQ connection
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	amqpChan, err = conn.Channel()
	if err != nil {
		log.Fatal(err)
	}
	defer amqpChan.Close()

	// Define queue
	_, err = amqpChan.QueueDeclare("ticket_queue", true, false, false, false, nil)

	// Start worker
	go startDBWorker()

	r := chi.NewRouter()
	r.Post("/buy", buyWithQueue)

	fmt.Println("Event-Driven System is ready! Port: 8080")
	fmt.Println("RabbitMQ Dashboard: http://localhost:15672")
	http.ListenAndServe(":8080", r)
}

func buyWithQueue(w http.ResponseWriter, r *http.Request) {
	// STEP 1: Redis Check
	newStock, _ := rdb.Decr(ctx, "ticket_stock:1").Result()
	if newStock < 0 {
		http.Error(w, "Tickets sold out!", http.StatusConflict)
		return
	}

	// STEP 2: Add message to queue (async)
	body := "1" // Selled ticket id
	err := amqpChan.PublishWithContext(ctx, "", "ticket_queue", false, false, amqp.Publishing{
		ContentType: "text/plain",
		Body:        []byte(body),
	})

	if err != nil {
		http.Error(w, "Queue error", 500)
		return
	}

	w.Write([]byte("Your process added to queue. Ticket is getting ready!"))
}

func startDBWorker() {
	msgs, _ := amqpChan.Consume("ticket_queue", "", true, false, false, false, nil)

	for d := range msgs {
		log.Printf("Message taken from queue: %s. DB updating...", d.Body)
		_, err := db.Exec("UPDATE tickets SET stock = stock - 1 WHERE id = 1")
		if err != nil {
			log.Println("DB Hatası:", err)
		}
	}
}

func initDBAndRedis() {
	db, _ = sql.Open("postgres", "postgres://user:password@localhost:5434/ticket_db?sslmode=disable")
	rdb = redis.NewClient(&redis.Options{Addr: "localhost:6379"})

	// Initialize DB with redis at start
	rdb.Set(ctx, "ticket_stock:1", 100, 0)
	db.Exec("UPDATE tickets SET stock = 100 WHERE id = 1")
}
