package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"
)

// User represents a sample user data structure.
type User struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// Product represents a sample product data structure.
type Product struct {
	ID    int     `json:"id"`
	Name  string  `json:"name"`
	Price float64 `json:"price"`
}

func usersHandler(w http.ResponseWriter, r *http.Request) {
	users := []User{
		{ID: 1, Name: "Alice"},
		{ID: 2, Name: "Bob"},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
	log.Println("Served request to /users")
}

func productsHandler(w http.ResponseWriter, r *http.Request) {
	products := []Product{
		{ID: 101, Name: "Laptop", Price: 1200.50},
		{ID: 102, Name: "Mouse", Price: 25.00},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(products)
	log.Println("Served request to /products")
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/users", usersHandler)
	mux.HandleFunc("/products", productsHandler)

	server := &http.Server{
		Addr:         ":3000",
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	log.Println("Mock API server starting on :3000...")
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
