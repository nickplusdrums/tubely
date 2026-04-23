package main

import (
	"fmt"
	"net/http"
	"io"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
	"errors"
	"path/filepath"
	"os"
	"strings"
	"mime"
	"crypto/rand"
	"encoding/base64"
)

func (cfg *apiConfig) handlerUploadThumbnail(w http.ResponseWriter, r *http.Request) {
	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find JWT", err)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT", err)
		return
	}


	fmt.Println("uploading thumbnail for video", videoID, "by user", userID)

	// TODO: implement the upload here
	
	const maxMemory = 10 << 20

	err = r.ParseMultipartForm(maxMemory)
	if err != nil {
		respondWithError(w, 400, "Could not parse", err)
		return
	}

	file, header, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, 500, "Failed to get file and header", err)
		return
	}

	defer file.Close()

	mediaType := header.Header.Get("Content-Type")
	parsedType, _, err := mime.ParseMediaType(mediaType)
	if err != nil || (parsedType != "image/png" && parsedType != "image/jpeg") {
		respondWithError(w, http.StatusBadRequest, "Wrong Media Type", err)
		return
	}

	var ext string

	parts := strings.Split(parsedType, "/")
	if len(parts) < 2 {
		respondWithError(w, http.StatusBadRequest, "Wrong Content Type", errors.New("Wrong Content Type"))
		return
	}
	switch parts[1] {
	case "png":
		ext = "png"
	case "jpeg":
		ext = "jpeg"
	case "gif":
		ext = "gif"
	default:
		respondWithError(w, http.StatusBadRequest, "Wrong Content Type", errors.New("Wrong Content Type"))
		return
	}
	
	key := make([]byte, 32)
	rand.Read(key)
	encoded := base64.RawURLEncoding.EncodeToString(key)


	fileName := fmt.Sprintf("%v.%v", encoded, ext)
	
	videoMD, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, 500, "Error retrieving video", err)
		return
	}

	if videoMD.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "Not authorized", err)
		return
	}
	
	fileURL := fmt.Sprintf("http://localhost:%v/assets/%v", cfg.port, fileName)
	diskPath := filepath.Join(cfg.assetsRoot, fileName)

	dst, err := os.Create(diskPath)
	if err != nil {
		respondWithError(w, 500, "Failed to create file", err)
		return
	}

	defer dst.Close()

	_, err = io.Copy(dst, file)
	if err != nil {
		respondWithError(w, 500, "Failed to copy file", err)
		return
	}

	videoMD.ThumbnailURL = &fileURL

	err = cfg.db.UpdateVideo(videoMD)
	if err != nil {
		respondWithError(w, 500, "Couldnt update video", err)
		return
	}

	respondWithJSON(w, http.StatusOK, videoMD)
}
