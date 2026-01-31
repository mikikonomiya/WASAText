package api

import (
	"encoding/json"
	"net/http"
	"wasatext/service/api/reqcontext"
	"wasatext/service/database"

	"github.com/julienschmidt/httprouter"
)

func (rt *_router) searchUsers(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	// 1. Get the query parameter "username" from the URL
	query := r.URL.Query().Get("username")

	// 2. Call the Database
	users, err := rt.db.SearchUsers(query)
	if err != nil {
		ctx.Logger.WithError(err).Error("database error in searchUsers")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// 3. Handle the "404 Not Found" requirement
	// If the user searched for a specific name (query != "") AND no results were found:
	if query != "" && len(users) == 0 {
		w.WriteHeader(http.StatusNotFound)
		// YAML says return a JSON error schema
		if err := json.NewEncoder(w).Encode(map[string]string{"error": "Username not found"}); err != nil {
			ctx.Logger.WithError(err).Error("error encoding response")
		}
		return
	}

	// 4. Send the Response (200 OK)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// If users is nil, make it an empty array so JSON is "[]" instead of "null"
	if users == nil {
		users = []database.User{}
	}

	if err := json.NewEncoder(w).Encode(users); err != nil {
		ctx.Logger.WithError(err).Error("error encoding response")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}
