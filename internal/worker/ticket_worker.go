package worker

import (
	"context"
	"log"
	"strconv"

	"github.com/DinVisel/EventDrivenTicketSystem/internal/domain"
	amqp "github.com/rabbitmq/amqp091-go"
)

type TicketWorker struct {
	repo     domain.TicketRepository
	amqpChan *amqp.Channel
}

func NewTicketWorker(repo domain.TicketRepository, ch *amqp.Channel) *TicketWorker {
	return &TicketWorker{
		repo:     repo,
		amqpChan: ch,
	}
}

func (w *TicketWorker) Start(ctx context.Context) {
	// Start listenning queue
	msgs, err := w.amqpChan.Consume(
		"ticket_queue",
		"",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("Queue could not listenned: %v", err)
	}

	log.Println("Worker is working, awaiting messages...")

	// Process messages in infinite loop
	for d := range msgs {
		ticketID, _ := strconv.Atoi(string(d.Body))

		// Update DB
		err := w.repo.UpdateStockDB(ctx, ticketID, 1)

		if err != nil {
			log.Printf("DB update error (TicketID %d): %v", ticketID, err)
			// We will send message to 'Retry Queue' later.
		} else {
			log.Printf("Ticket succesfully processed to DB: TicketID %d", ticketID)
		}
	}
}
