package main

import (
	"fmt"
	"net/http"
	"os"

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
		mem.SetClient("loadtest", 10000000, 10000000)
		s = mem
		fmt.Println("Using in-memory store")
	}
	handler := api.NewHandler(s)

	http.HandleFunc("/check", handler.Check)
	http.HandleFunc("/status/", handler.Status)
	http.HandleFunc("/config", handler.Config)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	fmt.Println("Rate limiter running on port " + port + "...")

	err := http.ListenAndServe(":"+port, nil)
	if err != nil {
		fmt.Println("Server error:", err)
	}
}
