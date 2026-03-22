package main

import (
	"fmt"
	"net/http"

	"github.com/whysooharsh/rate-limiter/store"
)

func main() {
	memStore := store.NewMemoryStore()
	memStore.AddClient("client1", 10, 1)
	memStore.AddClient("client25", 5, 1)

	fmt.Println("Server running on PORT 8000")

	err := http.ListenAndServe(":8000", nil)
	if err != nil {
		fmt.Println("Server error : ", err)
	}

}
