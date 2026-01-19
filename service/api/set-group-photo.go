package api

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"wasatext/service/api/reqcontext"
	"wasatext/service/database" // Needed for database.GenerateNewID

	"github.com/julienschmidt/httprouter"
)

// operationId: setGroupPhoto
func (rt *_router) setGroupPhoto(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {

	// 1. Auth
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	myID := strings.TrimPrefix(authHeader, "Bearer ")
	groupID := ps.ByName("groupId")

	// 2. Parse Multipart Form (Max 5MB)
	err := r.ParseMultipartForm(5 << 20)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// 3. Get File from Form
	file, _, err := r.FormFile("photo")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	defer file.Close()

	// 4. Generate Unique Filename
	// Note: We use "grp-" prefix to distinguish from message photos
	uniqueID, err := database.GenerateNewID()
	if err != nil {
		ctx.Logger.WithError(err).Error("id generation error")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	filename := "grp-" + uniqueID + ".png"

	// 5. Save File to Disk
	dstPath := filepath.Join("./doc/images", filename)
	_ = os.MkdirAll("./doc/images", 0755) // Ensure folder exists

	dst, err := os.Create(dstPath)
	if err != nil {
		ctx.Logger.WithError(err).Error("error creating file")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		ctx.Logger.WithError(err).Error("error saving file")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// 6. Update Database
	err = rt.db.SetGroupPhoto(groupID, myID, filename)

	// 7. Handle Errors
	if err != nil {
		// If DB update fails, we should delete the uploaded file to keep things clean
		_ = os.Remove(dstPath)

		if strings.Contains(err.Error(), "forbidden") {
			// As per your previous pattern, you might use 404 or 403.
			// The YAML says 403 for not a member, 404 for not found.
			// Our DB check bunches them. 403 Forbidden is a safe default.
			w.WriteHeader(http.StatusForbidden)
			return
		}
		ctx.Logger.WithError(err).Error("db error")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// 8. Success (204 No Content)
	w.WriteHeader(http.StatusNoContent)
}
