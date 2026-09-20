package api

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"

	"github.com/whysooharsh/rate-limiter/store"
)

type ConfigRequest struct {
	ClientID   string `json:"client_id"`
	MaxTokens  int    `json:"max_tokens"`
	RefillRate int    `json:"refill_rate"`
}

type HandlerOptions struct {
	TrustProxy    bool
	TrustClientID bool
	AdminAPIKey   string
}

type Handler struct {
	store   store.Store
	options HandlerOptions
}

func NewHandler(store store.Store, opts HandlerOptions) *Handler {
	return &Handler{store: store, options: opts}
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
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var req CheckRequest
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&req); err != nil && err != io.EOF {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	clientID := strings.TrimSpace(req.ClientID)
	if !h.options.TrustClientID || clientID == "" {
		clientID = h.getClientIP(r)
	}
	if clientID == "" {
		http.Error(w, "could not determine client identity", http.StatusBadRequest)
		return
	}
	allowed, currTok, maxTok := h.store.Allow(clientID)

	w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", maxTok))
	w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", currTok))

	w.Header().Set("Content-Type", "application/json")

	if allowed {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(CheckResponse{
			Allowed: true,
			Message: "request allowed",
		})
	} else {
		w.Header().Set("Retry-After", "1")
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
	id := strings.TrimPrefix(path, "/status/")
	if id == "" {
		http.Error(w, "client id is required", http.StatusBadRequest)
		return
	}
	currTok, maxTok, found := h.store.GetStatus(id)

	if !found {
		http.Error(w, "client not found", http.StatusNotFound)
		return
	}

	type response struct {
		CurrToken int
		MaxToken  int
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

	if h.options.AdminAPIKey == "" {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	providedKey := r.Header.Get("X-Admin-Key")
	if subtle.ConstantTimeCompare([]byte(providedKey), []byte(h.options.AdminAPIKey)) != 1 {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var req ConfigRequest
	dec := json.NewDecoder(r.Body)
	err := dec.Decode(&req)

	if err != nil || req.ClientID == "" {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if dec.Decode(&struct{}{}) != io.EOF {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.MaxTokens <= 0 {
		http.Error(w, "max tokens must be greater than 0", http.StatusBadRequest)
		return
	}

	if req.MaxTokens > 1000000000 {
		http.Error(w, "max tokens must not exceed 1000000000", http.StatusBadRequest)
		return
	}

	if req.RefillRate < 0 {
		http.Error(w, "refill rate must be non-negative", http.StatusBadRequest)
		return
	}

	if req.RefillRate > 1000000000 {
		http.Error(w, "refill rate must not exceed 1000000000", http.StatusBadRequest)
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

func (h *Handler) getClientIP(r *http.Request) string {

	if h.options.TrustProxy {
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			first, _, _ := strings.Cut(xff, ",")
			return strings.TrimSpace(first)
		}

	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}

	return host

}
