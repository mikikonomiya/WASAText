package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"wasatext/service/api/reqcontext"

	"github.com/julienschmidt/httprouter"
)

func (rt *_router) getUserProfile(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	// 1. get username from URL path (ex. /users/LUcia)
	username := ps.ByName("username")

	// 2. call the DB
	user, err := rt.db.GetUser(username)

	// 3. handle errors
	if err != nil {
		// if user not found -> 404
		if errors.Is(err, sql.ErrNoRows) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		// if database crush -> 500
		ctx.Logger.WithError(err).Error("database error in getUserProfile")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// 4. send the response 200 OK
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(user); err != nil {
		ctx.Logger.WithError(err).Error("error encoding response")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}
