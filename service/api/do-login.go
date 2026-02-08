package api

import (
	"encoding/json"
	"net/http"
	"regexp"
	"wasatext/service/api/reqcontext"

	"github.com/julienschmidt/httprouter"
)

// define what the JSON looks like
type loginRequest struct {
	Name string `json:"name"`
}

type loginResponse struct {
	Identifier string `json:"identifier"`
}

// handler function
func (rt *_router) doLogin(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {

	// 1) decode the request
	var req loginRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest) // return 400 Bad Request
		return
	}

	// 2) validate input (length, regex pattern with alphanumeic+dot+underscore)
	if len(req.Name) < 3 || len(req.Name) > 16 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if !regexp.MustCompile(`^[a-zA-Z0-9._]+$`).MatchString(req.Name) {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// 3) call the database
	id, err := rt.db.GetOrCreateUser(req.Name)
	if err != nil {
		// Log the error for debugging
		ctx.Logger.WithError(err).Error("database error in doLogin")
		// Return 500 Internal Server Error
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// 4) send the response (after getting the ID) = must convert back to json
	// create response structure
	resp := loginResponse{
		Identifier: id,
	}

	// Set headers
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated) // 201 Created (as per YAML)

	// Encode struct to JSON, translating
	err = json.NewEncoder(w).Encode(resp)
	if err != nil {
		ctx.Logger.WithError(err).Error("error encoding response")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}
