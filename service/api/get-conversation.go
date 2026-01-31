package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"wasatext/service/api/reqcontext"
	"wasatext/service/database" // Import for []database.Message

	"github.com/julienschmidt/httprouter"
)

// Matches operationId: getConversation
func (rt *_router) getConversation(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {

	// 1. Auth
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	myID := strings.TrimPrefix(authHeader, "Bearer ")

	// 2. Get Conversation ID from URL
	conversationID := ps.ByName("conversationId")

	// 3. Call DB
	messages, err := rt.db.GetConversationMessages(conversationID, myID)

	// 4. Handle Errors
	if err != nil {
		// If the error message matches what we wrote in Step 4
		if errors.Is(err, database.ErrConversationNotFound) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		ctx.Logger.WithError(err).Error("database error")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// 5. Success
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// Ensure empty list is "[]" not "null"
	if messages == nil {
		messages = []database.Message{}
	}

	if err := json.NewEncoder(w).Encode(messages); err != nil {
		ctx.Logger.WithError(err).Error("error encoding response")
	}
}
