package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/DinVisel/EventDrivenTicketSystem/internal/domain"
	amqp "github.com/rabbitmq/amqp091-go"
)

type ticketService struct {
	repo     domain.TicketRepository
	amqpChan *amqp.Channel
}

func NewTicketService(repo domain.TicketRepository, ch *amqp.Channel) domain.TicketService {
	return &ticketService{
		repo:     repo,
		amqpChan: ch,
	}
}

func (s *ticketService) Purchase(ctx context.Context, ticketID int) error {
	resourceID := fmt.Sprintf("ticket:%d", ticketID)

	//Lock the door with redlock
	mutex, err := s.repo.AcquireLock(ctx, resourceID)
	if err != nil {
		return fmt.Errorf("Process busy now. Please try again later: %w", err)
	}

	//Guarantee unlocking after process (even if there occurs an error)
	defer mutex.Unlock()

	// try decreasing quickly from Redis
	newStock, err := s.repo.DecrementStockCache(ctx, ticketID)
	if err != nil {
		return fmt.Errorf("Cache error: %w", err)
	}

	// if stock is empty (minus) reject immediately
	if newStock < 0 {
		return errors.New("Unfortunately tickets sold out")
	}

	// message to queue (async)
	// Do not make user wait until db update
	msg := fmt.Sprintf("%d", ticketID)
	err = s.amqpChan.PublishWithContext(ctx,
		"",
		"ticket_queue",
		false,
		false,
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte(msg),
		})

	if err != nil {
		// CRITICAL: if we cannot add to queue, we should return the place in redis. (Rollback)
		return fmt.Errorf("Queue error: %w", err)
	}
	return nil
}
