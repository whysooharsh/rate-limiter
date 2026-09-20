package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/whysooharsh/rate-limiter/api"
	"github.com/whysooharsh/rate-limiter/store"
)

func main() {
	_ = godotenv.Load()
	redisURL := os.Getenv("REDIS_URL")

	var s store.Store
	if redisURL != "" {
		s = store.NewRedisStore(redisURL)
		s.SetClient("client1", 10, 1)
		s.SetClient("client2", 5, 1)
		fmt.Println("Using Redis store")
	} else {
		mem := store.NewMemoryStore()
		mem.SetClient("client1", 10, 1)
		mem.SetClient("client2", 5, 1)
		s = mem
		fmt.Println("Using in-memory store")
	}
	handler := api.NewHandler(s, api.HandlerOptions{
		TrustProxy:    false,
		TrustClientID: false,
		AdminAPIKey:   os.Getenv("ADMIN_API_KEY"),
	})

	http.HandleFunc("/check", handler.Check)
	http.HandleFunc("/status/", handler.Status)
	http.HandleFunc("/config", handler.Config)
	http.HandleFunc("/baseline", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	fmt.Println("Rate limiter running on port " + port + "...")

	server := &http.Server{
		Addr:              ":" + port,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		<-ctx.Done()
		fmt.Println("shutting down...")
		shutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		server.Shutdown(shutCtx)
	}()

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		fmt.Println("Server error:", err)
	}
}
