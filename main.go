package main

import (
	"fmt"
	"net/http"

	"github.com/whysooharsh/rate-limiter/api"
	"github.com/whysooharsh/rate-limiter/store"
)

func main() {
	memStore := store.NewMemoryStore()
	memStore.AddClient("client1", 10, 1)
	memStore.AddClient("client2", 5, 1)

	handler := api.NewHandler(memStore)

	http.HandleFunc("/check", handler.Check)
	http.HandleFunc("/status/", handler.Status)
	http.HandleFunc("/config", handler.Config)
	fmt.Println("Rate limiter running on port 8080...")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		fmt.Println("Server error:", err)
	}
}
