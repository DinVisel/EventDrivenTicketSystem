package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/DinVisel/EventDrivenTicketSystem/internal/domain"
	"github.com/go-chi/chi/v5"
)

type TicketHandler struct {
	service domain.TicketService
}

func NewTicketHandler(s domain.TicketService) *TicketHandler {
	return &TicketHandler{service: s}
}

func (h *TicketHandler) BuyTicket(w http.ResponseWriter, r *http.Request) {
	// Take ticketID from url
	ticketIDStr := chi.URLParam(r, "ticketID")
	ticketID, _ := strconv.Atoi(ticketIDStr)

	// Call service layer
	err := h.service.Purchase(r.Context(), ticketID)

	// return response
	if err != nil {
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Your request has been queued!"})
}
