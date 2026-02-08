package api

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"wasatext/service/api/reqcontext"
	"wasatext/service/database"

	"github.com/julienschmidt/httprouter"
)

func (rt *_router) setMyPhoto(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {

	// 1. Authenticate (Get User ID)
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	userID := strings.TrimPrefix(authHeader, "Bearer ")

	// 2. Parse the Multipart Form
	// 10 << 20 means allow up to 10MB file size in memory
	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// 3. Get the file from the form
	file, _, err := r.FormFile("photo")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	defer file.Close()

	// 4. Create the file on disk
	// We will save it as "images/<userid>.png"
	// Ensure the directory exists first!
	const imagePath = "./data/images"
	_ = os.MkdirAll(imagePath, 0755) // Creates data/images folder if missing

	// we use a random suffix (or timestamp) to avoid browser caching issues when the photo changes.
	// Simple approach: userID + "_" + randomHex + ".png"
	rnd, _ := database.GenerateNewID()
	filename := userID + "_" + rnd + ".png"
	dstPath := filepath.Join(imagePath, filename)

	dst, err := os.Create(dstPath)
	if err != nil {
		ctx.Logger.WithError(err).Error("error creating file")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	// 5. Copy the uploaded bytes to the new file
	if _, err := io.Copy(dst, file); err != nil {
		ctx.Logger.WithError(err).Error("error saving file")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// 6. Update Database
	//  tell the DB: "User X now has photo 'X.png'"
	err = rt.db.SetUserPhoto(userID, filename)
	if err != nil {
		ctx.Logger.WithError(err).Error("database error")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// 7. Success
	w.WriteHeader(http.StatusNoContent)
}
