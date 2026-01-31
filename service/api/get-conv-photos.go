package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"wasatext/service/api/reqcontext"
	"wasatext/service/database"

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
		if errors.Is(err, database.ErrConversationNotFound) {
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

	// 4. Transform Filenames into URLs
	// If the list is empty, ensure we return "[]"
	photoURLs := make([]string, 0)

	// We assume your server runs on the configured port.
	// Ideally, this base URL should come from a config file.
	baseURL := "/images/"

	for _, fname := range filenames {
		fullURL := baseURL + fname
		photoURLs = append(photoURLs, fullURL)
	}

	// 5. Success
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(photoURLs); err != nil {
		ctx.Logger.WithError(err).Error("error encoding response")
	}
}
