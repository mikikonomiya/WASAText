package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"wasatext/service/api/reqcontext"

	"github.com/julienschmidt/httprouter"
)

// operationId: deleteConversation
func (rt *_router) deleteConversation(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {

	// 1. Auth
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	myID := strings.TrimPrefix(authHeader, "Bearer ")

	// 2. Get ID
	conversationID := ps.ByName("conversationId")

	// 3. Call DB (Update the timestamp)
	err := rt.db.ClearConversationHistory(conversationID, myID)

	// 4. Handle Errors
	if err != nil {
		if err.Error() == "conversation not found" {
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "Conversation not found"})
			return
		}
		ctx.Logger.WithError(err).Error("db error")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// 5. Success (204 No Content)
	w.WriteHeader(http.StatusNoContent)
}
