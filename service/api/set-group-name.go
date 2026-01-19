package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"wasatext/service/api/reqcontext"

	"github.com/julienschmidt/httprouter"
)

// operationId: setGroupName
func (rt *_router) setGroupName(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {

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
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// 3. Validate Length (3 to 25 chars)
	if len(body.Name) < 3 || len(body.Name) > 25 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// 4. Call DB
	err := rt.db.SetGroupName(groupID, myID, body.Name)

	// 5. Handle Errors
	if err != nil {
		errMsg := err.Error()

		if strings.Contains(errMsg, "forbidden") {
			// 403 Forbidden (or 404 if we want to hide existence)
			// Spec says 403 for not a member, 404 for not found.
			// Our DB logic bunches them, but 403 is safe here.
			w.WriteHeader(http.StatusForbidden)
			return
		}
		if strings.Contains(errMsg, "conflict") {
			// 409 Conflict
			w.WriteHeader(http.StatusConflict)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "Group name already taken"})
			return
		}

		ctx.Logger.WithError(err).Error("db error")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// 6. Success (204 No Content)
	w.WriteHeader(http.StatusNoContent)
}
