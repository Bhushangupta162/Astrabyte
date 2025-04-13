package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"github.com/Bhushangupta162/Astrabyte/db"
	"github.com/Bhushangupta162/Astrabyte/handlers"
	"github.com/Bhushangupta162/Astrabyte/middleware"
)

func main() {
	// Load environment variables
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	// Connect to DB
	db.Connect()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Router setup
	router := mux.NewRouter()

	// ✅ Health Check Route
	router.HandleFunc("/api/health", HealthCheckHandler).Methods("GET")

	// ✅ DB Connection Test Route
	router.HandleFunc("/api/db-check", func(w http.ResponseWriter, r *http.Request) {
		err := db.Pool.Ping(r.Context())
		if err != nil {
			http.Error(w, "Database not reachable", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"db connected"}`))
	}).Methods("GET")

	router.HandleFunc("/api/register", handlers.RegisterHandler).Methods("POST")

	router.HandleFunc("/api/login", handlers.LoginHandler).Methods("POST")

	router.Handle("/api/me", middleware.JWTAuthMiddleware(http.HandlerFunc(handlers.MeHandler))).Methods("GET")

	router.Handle("/api/upload", middleware.JWTAuthMiddleware(http.HandlerFunc(handlers.UploadHandler))).Methods("POST")

	router.Handle("/api/file/{id}", middleware.JWTAuthMiddleware(http.HandlerFunc(handlers.DownloadHandler))).Methods("GET")

	fmt.Println("🚀 Server running on port", port)
	log.Fatal(http.ListenAndServe(":"+port, router))
}

// Basic health check
func HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "ok"}`))
}
