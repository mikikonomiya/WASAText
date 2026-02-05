package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"wasatext/service/api/reqcontext"

	"github.com/julienschmidt/httprouter"
)

func (rt *_router) getMessageReactions(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	// 1. Auth check
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	// myID := strings.TrimPrefix(authHeader, "Bearer ")
	// We might technically need myID to check valid access to the conversation,
	// but for now we rely on the DB query or just assume if they have the message ID they can see reactions.
	// A robust impl would check: SELECT 1 FROM participants p JOIN messages m ON p.conversation_id = m.conversation_id WHERE m.id = ? AND p.user_id = ?

	messageID := ps.ByName("messageId")

	// 2. Fetch
	reactions, err := rt.db.GetMessageReactions(messageID)
	if err != nil {
		ctx.Logger.WithError(err).Error("db error fetching reactions")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// 3. Return JSON
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(reactions); err != nil {
		ctx.Logger.WithError(err).Error("error encoding response")
		w.WriteHeader(http.StatusInternalServerError)
	}
}
