package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"wasatext/service/api/reqcontext"

	"github.com/julienschmidt/httprouter"
)

// operationId: commentMessage
func (rt *_router) commentMessage(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {

	// 1. Auth
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	myID := strings.TrimPrefix(authHeader, "Bearer ")
	messageID := ps.ByName("messageId")

	// 2. Parse Body
	var body struct {
		Emoticon string `json:"emoticon"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	// DEBUG
	fmt.Printf("[DEBUG] React Request: MsgID=%s, User=%s, Emoji=%s\n", messageID, myID, body.Emoticon)

	// 3. Validation (simple length check)
	// The regex in YAML is complex '^\P{L}\s*$' (unicode non-letter),
	// but a length check covers the basics for now.
	if len(body.Emoticon) < 1 || len(body.Emoticon) > 16 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// 4. Call DB
	err := rt.db.ReactToMessage(messageID, myID, body.Emoticon)

	// 5. Handle Errors
	if err != nil {
		if err.Error() == "message not found or access denied" {
			w.WriteHeader(http.StatusNotFound)
			if err := json.NewEncoder(w).Encode(map[string]string{"error": "Message not found"}); err != nil {
				ctx.Logger.WithError(err).Error("error encoding response")
			}
			return
		}
		ctx.Logger.WithError(err).Error("db error")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// 6. Success (204 No Content)
	w.WriteHeader(http.StatusNoContent)
}
