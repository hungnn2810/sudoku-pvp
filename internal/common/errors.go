package common

import "errors"

// Sentinel errors used across the application.
var (
	ErrNotFound   = errors.New("not found")
	ErrUnauthorized = errors.New("unauthorized")
	ErrForbidden  = errors.New("forbidden")
	ErrConflict   = errors.New("conflict")
	ErrBadRequest = errors.New("bad request")
	ErrInternal   = errors.New("internal server error")
)
