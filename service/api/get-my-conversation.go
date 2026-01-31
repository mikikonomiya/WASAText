package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"wasatext/service/api/reqcontext"
	"wasatext/service/database"

	"github.com/julienschmidt/httprouter"
)

func (rt *_router) getMyConversations(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {

	// 1. Authenticate (Get User ID)
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	userID := strings.TrimPrefix(authHeader, "Bearer ")

	// 2. Call DB
	conversations, err := rt.db.GetMyConversations(userID)
	if err != nil {
		ctx.Logger.WithError(err).Error("database error getting conversations")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// 3. Handle Empty List
	// If the user has 0 chats, ensure we return "[]" not "null"
	if conversations == nil {
		conversations = []database.Conversation{}
	}

	// 4. Send Response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(conversations); err != nil {
		ctx.Logger.WithError(err).Error("error encoding response")
	}
}
