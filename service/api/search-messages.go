package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"wasatext/service/api/reqcontext"
	"wasatext/service/database" // For empty array initialization

	"github.com/julienschmidt/httprouter"
)

// operationId: searchMessages
func (rt *_router) searchMessages(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {

	// 1. Auth
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	myID := strings.TrimPrefix(authHeader, "Bearer ")

	// 2. Get Query Parameter
	// URL: /messages?textMessage=hello
	query := r.URL.Query().Get("textMessage")

	// 3. Validate
	if len(query) < 1 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// 4. Call DB
	messages, err := rt.db.SearchMessages(myID, query)
	if err != nil {
		ctx.Logger.WithError(err).Error("db error")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// 5. Success
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// Ensure empty array [] instead of null
	if messages == nil {
		messages = []database.Message{}
	}

	if err := json.NewEncoder(w).Encode(messages); err != nil {
		ctx.Logger.WithError(err).Error("error encoding response")
	}
}
