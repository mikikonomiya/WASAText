package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"wasatext/service/api/reqcontext"
	"wasatext/service/database"

	"github.com/julienschmidt/httprouter"
)

func (rt *_router) sendMessage(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {

	// 1. Auth
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	myID := strings.TrimPrefix(authHeader, "Bearer ")

	// 2. Params
	conversationID := ps.ByName("conversationId")

	// 3. Parse Body
	// YAML says: properties: textContent
	var body struct {
		TextContent string `json:"textContent"`
		ReplyTo     string `json:"replyTo"` // Optional
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// 4. Validate (Min 1, Max 1000)
	if len(body.TextContent) < 1 || len(body.TextContent) > 1000 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// 5. Call Database
	message, err := rt.db.SendMessage(conversationID, myID, body.TextContent, body.ReplyTo)
	if err != nil {
		if errors.Is(err, database.ErrConversationNotFound) {
			w.WriteHeader(http.StatusForbidden)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "You are not a member of this chat"})
			return
		}
		ctx.Logger.WithError(err).Error("db error")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	// 6. Success (200 OK)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(message); err != nil {
		ctx.Logger.WithError(err).Error("error encoding response")
	}
}
