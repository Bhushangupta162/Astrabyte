package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"

	"github.com/Bhushangupta162/Astrabyte/db"
	"github.com/Bhushangupta162/Astrabyte/middleware"
	"github.com/Bhushangupta162/Astrabyte/utils"
)

func UploadHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(middleware.UserIDKey)
	if userID == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse multipart form (max 50MB)
	err := r.ParseMultipartForm(50 << 20)
	if err != nil {
		http.Error(w, "Could not parse form", http.StatusBadRequest)
		return
	}

	// Read file
	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "File missing", http.StatusBadRequest)
		return
	}
	defer file.Close()

	fileBytes, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "Could not read file", http.StatusInternalServerError)
		return
	}

	fmt.Println("Encrypting file of size:", len(fileBytes), "bytes")

	// Encrypt file
	aesKey := []byte("1234567890abcdefgh1234567890abcd") // static AES key
	fmt.Println("AES key length:", len(aesKey))
	encryptedData, err := utils.Encrypt(fileBytes, aesKey)
	if err != nil {
		http.Error(w, "Encryption failed", http.StatusInternalServerError)
		return
	}

	// Upload to Lighthouse
	ipfsCID, err := uploadToLighthouse(encryptedData, header.Filename)
	if err != nil {
		http.Error(w, "Failed to upload to IPFS", http.StatusInternalServerError)
		return
	}

	// Calculate file size in MB
	sizeMB := float64(len(encryptedData)) / (1024 * 1024)

	// Save metadata and return file ID
	var fileID string
	err = db.Pool.QueryRow(r.Context(),
		`INSERT INTO files (user_id, filename, ipfs_cid, size_mb, encrypted)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id`,
		userID, header.Filename, ipfsCID, sizeMB, true,
	).Scan(&fileID)

	if err != nil {
		http.Error(w, "DB insert failed", http.StatusInternalServerError)
		return
	}

	// Return response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"id":       fileID,
		"cid":      ipfsCID,
		"filename": header.Filename,
		"size_mb":  fmt.Sprintf("%.2f", sizeMB),
	})
}

func uploadToLighthouse(encryptedData []byte, filename string) (string, error) {
	var b bytes.Buffer
	writer := multipart.NewWriter(&b)

	// Create the file part
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return "", err
	}
	_, err = part.Write(encryptedData)
	if err != nil {
		return "", err
	}

	writer.Close()

	// Correct Lighthouse endpoint
	req, err := http.NewRequest("POST", "https://node.lighthouse.storage/api/v0/add", &b)
	if err != nil {
		return "", err
	}

	// Correct header
	req.Header.Set("Authorization", "Bearer "+os.Getenv("LIGHTHOUSE_API_KEY"))
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Println("⬅️ Lighthouse response:", string(body))

	var result struct {
		Hash string `json:"Hash"`
		Name string `json:"Name"`
		Size string `json:"Size"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	if result.Hash == "" {
		return "", fmt.Errorf("upload failed, CID is empty")
	}

	return result.Hash, nil
}
