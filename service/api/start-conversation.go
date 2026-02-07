package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"wasatext/service/api/reqcontext"

	"github.com/julienschmidt/httprouter"
)

// Renamed function to match YAML operationId
func (rt *_router) startConversation(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {

	// 1. Auth
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	myID := strings.TrimPrefix(authHeader, "Bearer ")

	// 2. Parse Request
	var body struct {
		TargetUsername string `json:"targetUsername"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	// Validation
	if len(body.TargetUsername) < 3 || len(body.TargetUsername) > 16 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// 3. Call DB
	convo, created, err := rt.db.CreateOneToOneConversation(myID, body.TargetUsername)

	// 4. Handle Errors
	if err != nil {
		if err.Error() == "target user not found" {
			w.WriteHeader(http.StatusNotFound)
			if err := json.NewEncoder(w).Encode(map[string]string{"error": "Target user not found"}); err != nil {
				ctx.Logger.WithError(err).Error("error encoding response")
			}
			return
		}
		ctx.Logger.WithError(err).Error("db error")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// 5. Success Responses
	w.Header().Set("Content-Type", "application/json")

	if created {
		// Status 201: Return Full Object but map ID -> conversationId
		w.WriteHeader(http.StatusCreated)
		// We create a temporary struct or map to override the JSON tag
		resp := struct {
			ConversationId string `json:"conversationId"`
			Name           string `json:"name,omitempty"`
			LastMessage    string `json:"last_message"`
			UpdatedAt      string `json:"updated_at"`
		}{
			ConversationId: convo.ID,
			Name:           convo.Name,
			LastMessage:    convo.LastMessage,
			UpdatedAt:      convo.UpdatedAt,
		}
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			ctx.Logger.WithError(err).Error("error encoding response")
		}
	} else {
		// Status 200: Return Full Object
		w.WriteHeader(http.StatusOK)
		resp := struct {
			ConversationId string `json:"conversationId"`
			Name           string `json:"name,omitempty"`
			LastMessage    string `json:"last_message"`
			UpdatedAt      string `json:"updated_at"`
		}{
			ConversationId: convo.ID,
			Name:           convo.Name,
			LastMessage:    convo.LastMessage,
			UpdatedAt:      convo.UpdatedAt,
		}
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			ctx.Logger.WithError(err).Error("error encoding response")
		}
	}
}
