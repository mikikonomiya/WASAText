package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"wasatext/service/api/reqcontext"

	"github.com/julienschmidt/httprouter"
)

// operationId: addToGroup
func (rt *_router) addToGroup(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {

	// 1. Auth
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	myID := strings.TrimPrefix(authHeader, "Bearer ")
	groupID := ps.ByName("groupId")

	// 2. Parse Body
	var body struct {
		Username string `json:"username"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// 3. Validation
	if len(body.Username) < 3 || len(body.Username) > 16 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// 4. Call DB
	err := rt.db.AddUserToGroup(groupID, myID, body.Username)

	// 5. Handle Errors
	if err != nil {
		errMsg := err.Error()

		if strings.Contains(errMsg, "forbidden") {
			// 403 Forbidden
			w.WriteHeader(http.StatusForbidden)
			return
		}
		if strings.Contains(errMsg, "target user not found") {
			// 404 Not Found
			w.WriteHeader(http.StatusNotFound)
			if err := json.NewEncoder(w).Encode(map[string]string{"error": "User not found"}); err != nil {
				ctx.Logger.WithError(err).Error("error encoding response")
			}
			return
		}
		if strings.Contains(errMsg, "conflict") {
			// 409 Conflict
			w.WriteHeader(http.StatusConflict)
			if err := json.NewEncoder(w).Encode(map[string]string{"error": "User already in group"}); err != nil {
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
