package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"wasatext/service/api/reqcontext"
	"wasatext/service/database"

	"github.com/julienschmidt/httprouter"
)

// operationId: getAllStarredMessages
func (rt *_router) getAllStarredMessages(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {

	// 1. Auth
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	myID := strings.TrimPrefix(authHeader, "Bearer ")

	// 2. Call DB
	messages, err := rt.db.GetAllStarredMessages(myID)

	// 3. Handle Errors
	if err != nil {
		ctx.Logger.WithError(err).Error("db error")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// 4. Success
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if messages == nil {
		messages = []database.Message{}
	}

	_ = json.NewEncoder(w).Encode(messages)
}
