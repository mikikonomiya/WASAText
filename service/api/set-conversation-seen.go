package api

import (
	"net/http"
	"strings"
	"wasatext/service/api/reqcontext"

	"github.com/julienschmidt/httprouter"
)

// operationId: setConversationSeen
func (rt *_router) setConversationSeen(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {

	// 1. Auth
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	myID := strings.TrimPrefix(authHeader, "Bearer ")

	// 2. Get ID
	conversationID := ps.ByName("conversationId")

	// 3. Call DB (Reuse ClearConversationHistory which updates last_seen to NOW)
	err := rt.db.ClearConversationHistory(conversationID, myID)

	// 4. Handle Errors
	if err != nil {
		ctx.Logger.WithError(err).Error("db error")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// 5. Success (204 No Content)
	w.WriteHeader(http.StatusNoContent)
}
