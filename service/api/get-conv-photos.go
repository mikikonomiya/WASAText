package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"wasatext/service/api/reqcontext"

	"github.com/julienschmidt/httprouter"
)

// operationId: getConvPhotos
func (rt *_router) getConvPhotos(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {

	// 1. Auth
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	myID := strings.TrimPrefix(authHeader, "Bearer ")
	conversationID := ps.ByName("conversationId")

	// 2. Call DB
	filenames, err := rt.db.GetConversationPhotos(conversationID, myID)

	// 3. Handle Errors
	if err != nil {
		if err.Error() == "conversation not found or access denied" {
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "Conversation not found"})
			return
		}
		ctx.Logger.WithError(err).Error("db error")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// 4. Transform Filenames into URLs
	// If the list is empty, ensure we return "[]"
	photoURLs := make([]string, 0)

	// We assume your server runs on localhost:3000.
	// Ideally, this base URL should come from a config file.
	baseURL := "http://localhost:3000/images/"

	for _, fname := range filenames {
		fullURL := baseURL + fname
		photoURLs = append(photoURLs, fullURL)
	}

	// 5. Success
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(photoURLs)
}
