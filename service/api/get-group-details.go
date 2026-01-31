package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"wasatext/service/api/reqcontext"

	"github.com/julienschmidt/httprouter"
)

// operationId: getGroupDetails
func (rt *_router) getGroupDetails(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {

	// 1. Auth
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	myID := strings.TrimPrefix(authHeader, "Bearer ")
	groupID := ps.ByName("groupId")

	// 2. Call DB
	details, err := rt.db.GetGroupDetails(groupID, myID)

	// 3. Handle Errors
	if err != nil {
		if err.Error() == "group not found or access denied" {
			w.WriteHeader(http.StatusNotFound)
			if err := json.NewEncoder(w).Encode(map[string]string{"error": "Group not found"}); err != nil {
				ctx.Logger.WithError(err).Error("error encoding response")
			}
			return
		}
		ctx.Logger.WithError(err).Error("db error")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// 4. Format Photo URL
	// If PhotoID is empty, you might want to send a default image URL or keep it empty string.
	if details.PhotoID != "" {
		details.PhotoID = "/images/" + details.PhotoID
	}

	// 5. Success
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(details); err != nil {
		ctx.Logger.WithError(err).Error("error encoding response")
	}
}
