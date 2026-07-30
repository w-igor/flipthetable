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

var allowedUploadExt = map[string]bool{
	".jpg":  true,
	".jpeg": true,
	".png":  true,
	".webp": true,
	".gif":  true,
}

func handleUploadPhoto(w http.ResponseWriter, r *http.Request) {
	if _, ok := userIDFromContext(r.Context()); !ok {
		writeError(w, http.StatusUnauthorized, "Brak autoryzacji")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		writeError(w, http.StatusBadRequest, "Plik jest za duży (limit 5 MB)")
		return
	}

	file, header, err := r.FormFile("photo")
	if err != nil {
		writeError(w, http.StatusBadRequest, "Brak pliku zdjęcia")
		return
	}
	defer file.Close()

	ext := filepath.Ext(header.Filename)
	if !allowedUploadExt[ext] {
		writeError(w, http.StatusBadRequest, "Dozwolone formaty: JPG, PNG, WEBP, GIF")
		return
	}

	nameBytes := make([]byte, 16)
	if _, err := rand.Read(nameBytes); err != nil {
		writeError(w, http.StatusInternalServerError, "Nie udało się zapisać pliku")
		return
	}
	filename := hex.EncodeToString(nameBytes) + ext

	if err := os.MkdirAll(uploadDir, 0o755); err != nil {
		writeError(w, http.StatusInternalServerError, "Nie udało się zapisać pliku")
		return
	}

	dst, err := os.Create(filepath.Join(uploadDir, filename))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Nie udało się zapisać pliku")
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		writeError(w, http.StatusInternalServerError, "Nie udało się zapisać pliku")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{
		"url": "http://" + r.Host + "/uploads/" + filename,
	})
}
