package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/DinVisel/EventDrivenTicketSystem/internal/handler"
	"github.com/DinVisel/EventDrivenTicketSystem/internal/repository"
	"github.com/DinVisel/EventDrivenTicketSystem/internal/service"
	"github.com/DinVisel/EventDrivenTicketSystem/internal/worker"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	_ "github.com/lib/pq"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
)

func main() {
	// Infanstructure connections
	ctx := context.Background()

	// PostgreSQL
	db, err := sql.Open("postgres", "postgres://user:password@localhost:5434/ticket_db?sslmode=disable")
	if err != nil {
		log.Fatal("DB connection error: ", err)
	}

	// Redis
	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})

	// RabbitMQ
	amqpConn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		log.Fatal("RabbitMQ connection error: ", err)
	}
	amqpChan, _ := amqpConn.Channel()
	amqpChan.QueueDeclare("ticket_queue", true, false, false, false, nil)

	// Connect layers with each other
	repo := repository.NewTicketRepository(db, rdb)
	svc := service.NewTicketService(repo, amqpChan)
	hdl := handler.NewTicketHandler(svc)
	wrk := worker.NewTicketWorker(repo, amqpChan)

	// Initial settings
	// Sync db and redis
	repo.SyncCacheWithDB(ctx, 100)

	// Start worker in background
	go wrk.Start(ctx)

	// Routing
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Post("/buy/{ticketID}", hdl.BuyTicket)

	// Start server with graceful shutdown
	server := &http.Server{Addr: ":8080", Handler: r}

	go func() {
		fmt.Println("Ultra-Fast Ticket System is live! Port: 8080")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Listen: %s\n", err)
		}
	}()

	// Wait for closing signal
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	fmt.Println("System shutting down...")

	// Wait 5 seconds to close connections safely
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	server.Shutdown(shutdownCtx)
	db.Close()
	amqpChan.Close()

	fmt.Println("System terminated succesfully!")

}
