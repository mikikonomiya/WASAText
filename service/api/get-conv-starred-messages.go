package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"wasatext/service/api/reqcontext"
	"wasatext/service/database"

	"github.com/julienschmidt/httprouter"
)

// operationId: getConvStarredMessages
func (rt *_router) getConvStarredMessages(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {

	// 1. Auth
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	myID := strings.TrimPrefix(authHeader, "Bearer ")
	conversationID := ps.ByName("conversationId")

	// 2. Call DB
	messages, err := rt.db.GetStarredMessages(conversationID, myID)

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

	// 4. Success
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if messages == nil {
		messages = []database.Message{}
	}

	_ = json.NewEncoder(w).Encode(messages)
}
