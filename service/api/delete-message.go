package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"wasatext/service/api/reqcontext"

	"github.com/julienschmidt/httprouter"
)

// operationId: deleteMessage
func (rt *_router) deleteMessage(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {

	// 1. Auth
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	myID := strings.TrimPrefix(authHeader, "Bearer ")
	messageID := ps.ByName("messageId")

	// 2. Get Scope Query Param
	scope := r.URL.Query().Get("scope")
	// YAML Validation: enum [me, everyone]
	if scope != "me" && scope != "everyone" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// 3. Call DB
	err := rt.db.DeleteMessage(messageID, myID, scope)

	// 4. Handle Errors
	if err != nil {
		errMsg := err.Error()
		if errMsg == "message not found" {
			w.WriteHeader(http.StatusNotFound)
			if err := json.NewEncoder(w).Encode(map[string]string{"error": "Message not found"}); err != nil {
				ctx.Logger.WithError(err).Error("error encoding response")
			}
			return
		}
		if strings.Contains(errMsg, "forbidden") {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		ctx.Logger.WithError(err).Error("db error")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// 5. Success (204 No Content)
	w.WriteHeader(http.StatusNoContent)
}
