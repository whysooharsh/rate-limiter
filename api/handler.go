package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/whysooharsh/rate-limiter/store"
)

type ConfigRequest struct {
	ClientID   string `json:"client_id"`
	MaxTokens  int    `json:"max_tokens"`
	RefillRate int    `json:"refill_rate"`
}

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
	json.NewDecoder(r.Body).Decode(&req)
	clientID := req.ClientID

	if clientID == "" {
		clientID = getClientIP(r)
	}

	allowed := h.store.Allow(clientID)

	currTok, maxTok := h.store.GetStatus(clientID)

	w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", maxTok))
	w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", currTok))
	w.Header().Set("Retry-After", "1")

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

func (h *Handler) Status(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	path := r.URL.Path
	id := path[len("/status/"):]

	type response struct {
		CurrToken int
		MaxToken  int
	}
	currTok, maxTok := h.store.GetStatus(id)

	if currTok == 0 && maxTok == 0 {
		http.Error(w, "client not found", http.StatusNotFound)
		return
	}

	res := response{CurrToken: currTok, MaxToken: maxTok}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

func (h *Handler) Config(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ConfigRequest
	err := json.NewDecoder(r.Body).Decode(&req)

	if err != nil || req.ClientID == "" {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	h.store.SetClient(req.ClientID, req.MaxTokens, req.RefillRate)

	w.Header().Set("Content-Type", "application/json")

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(ConfigRequest{
		ClientID:   req.ClientID,
		MaxTokens:  req.MaxTokens,
		RefillRate: req.RefillRate,
	})

}

func getClientIP(r *http.Request) string {
	ip := r.Header.Get("X-Forwarded-For")
	if ip == "" {
		ip = r.RemoteAddr
	}
	return ip
}
