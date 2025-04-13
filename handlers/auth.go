package handlers

import (
	"encoding/json"
	"net/http"

	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"time"
	"os"
	"github.com/Bhushangupta162/Astrabyte/middleware"
	"github.com/Bhushangupta162/Astrabyte/db"
	// "github.com/Bhushangupta162/Astrabyte/models"
	"golang.org/x/crypto/bcrypt"
)

type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil || req.Email == "" || req.Password == "" {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Error hashing password", http.StatusInternalServerError)
		return
	}

	// Save user
	_, err = db.Pool.Exec(r.Context(),
		`INSERT INTO users (email, password_hash) VALUES ($1, $2)`,
		req.Email, string(hashedPassword),
	)
	if err != nil {
		http.Error(w, "Email already exists or DB error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(`{"status":"user registered"}`))
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil || req.Email == "" || req.Password == "" {
		http.Error(w, "Invalid login request", http.StatusBadRequest)
		return
	}

	// Fetch user from DB
	var storedHash, userID string
	err = db.Pool.QueryRow(r.Context(),
		`SELECT id, password_hash FROM users WHERE email = $1`, req.Email).
		Scan(&userID, &storedHash)

	if err != nil {
		http.Error(w, "Invalid email or password", http.StatusUnauthorized)
		return
	}

	// Check password
	err = bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(req.Password))
	if err != nil {
		http.Error(w, "Invalid email or password", http.StatusUnauthorized)
		return
	}

	// Create JWT token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID,
		"email":   req.Email,
		"exp":     time.Now().Add(time.Hour * 24).Unix(), // expires in 1 day
	})

	// Sign token with secret
	secret := os.Getenv("JWT_SECRET")
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		http.Error(w, "Error generating token", http.StatusInternalServerError)
		return
	}

	// Send token
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(fmt.Sprintf(`{"token": "%s"}`, tokenString)))
}

func MeHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(middleware.UserIDKey)
	if userID == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Optional: Fetch more info from DB using userID
	response := fmt.Sprintf(`{"user_id": "%s"}`, userID)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(response))
}