package response

import (
	"errors"
	"net/http"
)

type AppError struct {
	Code     string `json:"code"`
	Message  string `json:"message"`
	Detail   string `json:"detail,omitempty"`
	Resource string `json:"resource,omitempty"`
}

func (e *AppError) Error() string { return e.Message }

func NewAppError(code, message string) *AppError { return &AppError{Code: code, Message: message} }
func NewDetailedError(code, message, detail string) *AppError {
	return &AppError{Code: code, Message: message, Detail: detail}
}

func IsAppError(err error) (*AppError, bool) {
	var app *AppError
	if errors.As(err, &app) {
		return app, true
	}
	return nil, false
}

func HTTPStatusFromError(err error) int {
	app, ok := IsAppError(err)
	if !ok {
		return http.StatusInternalServerError
	}
	switch app.Code {
	case CodeInvalidArgument, CodeManifestInvalid, CodeExecutionStateInvalid, CodePluginStateInvalid:
		return http.StatusBadRequest
	case CodeIdempotencyConflict:
		return http.StatusConflict
	case CodeUnauthorized:
		return http.StatusUnauthorized
	case CodeForbidden:
		return http.StatusForbidden
	case CodeNotFound, CodePluginNotFound, CodeExecutionNotFound:
		return http.StatusNotFound
	case CodeQueueFull:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}
