package main

import (
	"crypto/rand"
	"encoding/hex"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

const uploadDir = "uploads"
const maxUploadSize = 5 << 20 // 5 MB

// allowedUploadExt specifies which file extensions are accepted for photo uploads.
var allowedUploadExt = map[string]bool{
	".jpg":  true,
	".jpeg": true,
	".png":  true,
	".webp": true,
	".gif":  true,
}

// handleUploadPhoto processes a multipart form file upload, validates the file type/size,
// stores it with a random name, and returns the URL for serving the uploaded image.
func handleUploadPhoto(w http.ResponseWriter, r *http.Request) {
	// Require authentication
	if _, ok := userIDFromContext(r.Context()); !ok {
		writeError(w, http.StatusUnauthorized, "Brak autoryzacji")
		return
	}

	// Limit request body size to prevent abuse
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		writeError(w, http.StatusBadRequest, "Plik jest za duży (limit 5 MB)")
		return
	}

	// Extract the "photo" form field
	file, header, err := r.FormFile("photo")
	if err != nil {
		writeError(w, http.StatusBadRequest, "Brak pliku zdjęcia")
		return
	}
	defer file.Close()

	// Validate file extension
	ext := filepath.Ext(header.Filename)
	if !allowedUploadExt[ext] {
		writeError(w, http.StatusBadRequest, "Dozwolone formaty: JPG, PNG, WEBP, GIF")
		return
	}

	// Generate a random filename to prevent collisions and avoid exposing original names
	nameBytes := make([]byte, 16)
	if _, err := rand.Read(nameBytes); err != nil {
		writeError(w, http.StatusInternalServerError, "Nie udało się zapisać pliku")
		return
	}
	filename := hex.EncodeToString(nameBytes) + ext

	// Create uploads directory if it doesn't exist
	if err := os.MkdirAll(uploadDir, 0o755); err != nil {
		writeError(w, http.StatusInternalServerError, "Nie udało się zapisać pliku")
		return
	}

	// Create the destination file
	dst, err := os.Create(filepath.Join(uploadDir, filename))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Nie udało się zapisać pliku")
		return
	}
	defer dst.Close()

	// Copy uploaded file to disk
	if _, err := io.Copy(dst, file); err != nil {
		writeError(w, http.StatusInternalServerError, "Nie udało się zapisać pliku")
		return
	}

	// Return the URL where the file can be accessed
	writeJSON(w, http.StatusCreated, map[string]string{
		"url": "http://" + r.Host + "/uploads/" + filename,
	})
}
