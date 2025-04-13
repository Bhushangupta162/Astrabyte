package handlers

import (
	"fmt"
	"io"
	"net/http"

	"github.com/Bhushangupta162/Astrabyte/db"
	"github.com/Bhushangupta162/Astrabyte/middleware"
	"github.com/Bhushangupta162/Astrabyte/utils"
	"github.com/gorilla/mux"
)

func DownloadHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(middleware.UserIDKey)
	if userID == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	fileID := mux.Vars(r)["id"]

	// Lookup file by ID and check ownership
	var cid, filename string
	err := db.Pool.QueryRow(r.Context(),
		`SELECT ipfs_cid, filename FROM files WHERE id = $1 AND user_id = $2`,
		fileID, userID,
	).Scan(&cid, &filename)

	if err != nil {
		http.Error(w, "File not found or access denied", http.StatusNotFound)
		return
	}

	// Download encrypted data from IPFS
	ipfsURL := fmt.Sprintf("https://gateway.lighthouse.storage/ipfs/%s", cid)
	resp, err := http.Get(ipfsURL)
	if err != nil || resp.StatusCode != 200 {
		http.Error(w, "Failed to fetch file from IPFS", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	encryptedData, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "Failed to read IPFS response", http.StatusInternalServerError)
		return
	}

	// Decrypt the file
	aesKey := []byte("1234567890abcdefgh1234567890abcd") // same key used for encryption
	decryptedData, err := utils.Decrypt(encryptedData, aesKey)
	if err != nil {
		http.Error(w, "Decryption failed", http.StatusInternalServerError)
		return
	}

	// Return file
	w.Header().Set("Content-Disposition", "attachment; filename="+filename)
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(decryptedData)
}
