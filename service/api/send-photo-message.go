package api

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"wasatext/service/api/reqcontext"

	"github.com/julienschmidt/httprouter"
)

// operationId: sendPhotoMessage
func (rt *_router) sendPhotoMessage(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {

	// 1. Auth
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	myID := strings.TrimPrefix(authHeader, "Bearer ")
	conversationID := ps.ByName("conversationId")

	// 2. Parse Multipart Form (Max 10MB)
	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// 3. Get File from Form
	file, _, err := r.FormFile("photo")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	defer file.Close()

	// 4. Generate a unique filename
	// We can't use the db helper directly here because it is private.
	// Option A: Make `generateNewID` public in database package (Capital G).
	// Option B: Just use a simple UUID here.
	// Let's assume you rename the helper in database.go to "GenerateNewID" (Capital G).
	// If not, just implement a small random string generator here.

	// Quick random generator for filename:
	// (Ideally, move `generateNewID` to a shared `utils` package or make it public `database.GenerateNewID`)
	photoID := "img-" + myID + "-" + conversationID // Simple unique-ish name for now
	// BETTER: If you can, update database.go to make GenerateNewID public!

	filename := photoID + ".png"

	// 5. Save File to Disk
	dstPath := filepath.Join("./doc/images", filename)
	_ = os.MkdirAll("./doc/images", 0755) // Ensure folder exists

	dst, err := os.Create(dstPath)
	if err != nil {
		ctx.Logger.WithError(err).Error("error creating file")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		ctx.Logger.WithError(err).Error("error saving file")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// 6. Update Database
	err = rt.db.SendPhotoMessage(conversationID, myID, filename)

	// 7. Handle Errors
	if err != nil {
		if err.Error() == "conversation not found or access denied" {
			// Clean up the file if DB failed!
			_ = os.Remove(dstPath)
			w.WriteHeader(http.StatusNotFound)
			if err := json.NewEncoder(w).Encode(map[string]string{"error": "Conversation not found"}); err != nil {
				ctx.Logger.WithError(err).Error("error encoding response")
			}
			return
		}
		ctx.Logger.WithError(err).Error("db error")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// 8. Success (204 No Content)
	w.WriteHeader(http.StatusNoContent)
}
