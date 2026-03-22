package api

import (
	"encoding/json"
	"net/http"

	"github.com/whysooharsh/rate-limiter/store"
)

type Handler struct {
	store *store.MemoryStore
}

func NewHandler(store *store.MemoryStore) *Handler {
	return &Handler{store: store}
}

type CheckRequest struct {
	ClientID string `json:"client_id"`
}

type CheckResponse struct {
	Allowed bool   `json:"allowed"`
	Message string `json:"message"`
}

func (h *Handler) Check(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CheckRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil || req.ClientID == "" {
		http.Error(w, "invalid request body", http.StatusBadRequest)
	}

	allowed := h.store.Allow(req.ClientID)

	w.Header().Set("Content-Type", "application/json")

	if allowed {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(CheckResponse{
			Allowed: true,
			Message: "request allowed",
		})
	} else {
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(CheckResponse{
			Allowed: false,
			Message: "rate limit reached",
		})
	}
}
