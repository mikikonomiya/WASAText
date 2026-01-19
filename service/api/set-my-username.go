package api

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"
	"wasatext/service/api/reqcontext"

	"github.com/julienschmidt/httprouter"
)

func (rt *_router) setMyUserName(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {

	// 1. Get the User ID (Authentication), we extract it from the Authorization header: "Bearer <userid>"
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	// Extract the ID (everything after "Bearer ")
	myID := strings.TrimPrefix(authHeader, "Bearer ")

	// 2. Parse the Request Body (The new username)
	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// 3. Validate the Input (Regex/Length)
	if len(body.Name) < 3 || len(body.Name) > 16 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if !regexp.MustCompile(`^[a-zA-Z0-9._]+$`).MatchString(body.Name) {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// 4. Call the Database
	err := rt.db.SetUsername(myID, body.Name)

	// 5. Handle Errors (The 409 Conflict logic)
	if err != nil {
		// Check if the error string mentions the unique constraint
		// SQLite error for duplicates usually contains "UNIQUE constraint failed"
		if strings.Contains(err.Error(), "UNIQUE constraint failed") ||
			strings.Contains(err.Error(), "constraint failed") {
			w.WriteHeader(http.StatusConflict) // 409 Conflict
			return
		}

		// Other errors (e.g., db crash)
		ctx.Logger.WithError(err).Error("database error in setMyUserName")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// 6. Success (204 No Content)
	w.WriteHeader(http.StatusNoContent)
}
