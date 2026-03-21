package v1

import "net/http"

var (
	ErrBadRequest          = NewAppError(400, http.StatusBadRequest, "Bad Request")
	ErrUnauthorized        = NewAppError(401, http.StatusUnauthorized, "Unauthorized")
	ErrNotFound            = NewAppError(404, http.StatusNotFound, "Not Found")
	ErrInternalServerError = NewAppError(500, http.StatusInternalServerError, "Internal Server Error")

	ErrEmailAlreadyInUse = NewAppError(10_001, http.StatusBadRequest, "The email is already in use.")
)
