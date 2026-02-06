package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"wasatext/service/api/reqcontext"

	"github.com/julienschmidt/httprouter"
)

// operationId: forwardMessage
func (rt *_router) forwardMessage(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {

	// 1. Auth
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	myID := strings.TrimPrefix(authHeader, "Bearer ")
	// 2. Parse Body
	var body struct {
		SourceMessageID           string `json:"sourceMessageId"`
		DestinationConversationID string `json:"destinationConversationId"`
	}
	// Decode
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// 3. Validation
	if body.SourceMessageID == "" || body.DestinationConversationID == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// 4. Forward
	err := rt.db.ForwardMessage(myID, body.SourceMessageID, body.DestinationConversationID)
	if err != nil {
		// Handle specific errors
		errMsg := err.Error()
		if strings.Contains(errMsg, "not found") || strings.Contains(errMsg, "access denied") {
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "Message or conversation not found"})
			return
		}
		ctx.Logger.WithError(err).Error("db error forwarding message")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// 5. Success (204 No Content)
	w.WriteHeader(http.StatusNoContent)
}
