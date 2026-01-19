package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"wasatext/service/api/reqcontext"

	"github.com/julienschmidt/httprouter"
)

// operationId: createGroup
func (rt *_router) createGroup(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {

	// 1. Auth
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	myID := strings.TrimPrefix(authHeader, "Bearer ")

	// 2. Parse Body
	var body struct {
		GroupName       string   `json:"groupName"`
		MemberUsernames []string `json:"memberUsernames"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// 3. Validate
	if len(body.GroupName) < 3 || len(body.GroupName) > 25 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if len(body.MemberUsernames) < 1 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// 4. Call DB
	group, err := rt.db.CreateGroup(body.GroupName, myID, body.MemberUsernames)
	if err != nil {
		ctx.Logger.WithError(err).Error("db error")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// 5. Success (200 OK)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(group)
}
