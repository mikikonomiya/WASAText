package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"wasatext/service/api/reqcontext"

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

	// 5. Call DB
	msg, err := rt.db.SendMessage(conversationID, myID, body.TextContent)
	if err != nil {
		if err.Error() == "conversation not found or access denied" {
			w.WriteHeader(http.StatusNotFound)
			if err := json.NewEncoder(w).Encode(map[string]string{"error": "Conversation not found"}); err != nil {
				ctx.Logger.WithError(err).Error("error encoding response")
			}
			return
		}
		ctx.Logger.WithError(err).Error("db error")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// 6. Success (200 OK)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(msg); err != nil {
		ctx.Logger.WithError(err).Error("error encoding response")
	}
}
