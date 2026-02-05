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
	destConvID := ps.ByName("conversationId")

	// 2. Parse Body
	var body struct {
		SourceMessageIDs []string `json:"sourceMessageIds"`
	}
	// Decode
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// 3. Loop and Forward
	// If one fails, we log it but continue (best effort), or fail all?
	// Best effort seems appropriate for UI actions, or fail fast.
	// Let's go with "Fail Fast" for simplicity of error reporting.
	for _, msgID := range body.SourceMessageIDs {
		err := rt.db.ForwardMessage(myID, msgID, destConvID)
		if err != nil {
			// Handle specific errors
			errMsg := err.Error()
			if strings.Contains(errMsg, "not found") || strings.Contains(errMsg, "access denied") {
				w.WriteHeader(http.StatusNotFound)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "Message " + msgID + ": " + errMsg})
				// Stop processing
				return
			}
			// rt.db.ForwardMessage(myID, msgID, destConvID) // retry? No.
			ctx.Logger.WithError(err).Error("db error forwarding message " + msgID)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	// 5. Success (204 No Content)
	w.WriteHeader(http.StatusNoContent)
}
