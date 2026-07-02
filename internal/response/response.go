package response

import (
	"encoding/json"
	"net/http"
)

type Envelope struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Data      any    `json:"data,omitempty"`
	Error     any    `json:"error,omitempty"`
	RequestID string `json:"request_id,omitempty"`
}

func Success(w http.ResponseWriter, requestID string, data any) {
	writeJSON(w, http.StatusOK, Envelope{Code: CodeOK, Message: "ok", Data: data, RequestID: requestID})
}

func Created(w http.ResponseWriter, requestID string, data any) {
	writeJSON(w, http.StatusCreated, Envelope{Code: CodeOK, Message: "created", Data: data, RequestID: requestID})
}

func Fail(w http.ResponseWriter, requestID string, err error) {
	status := HTTPStatusFromError(err)
	app, ok := IsAppError(err)
	if !ok {
		app = NewAppError(CodeInternalError, "internal error")
	}
	writeJSON(w, status, Envelope{Code: app.Code, Message: app.Message, Error: app, RequestID: requestID})
}

func Page(w http.ResponseWriter, requestID string, data any) { Success(w, requestID, data) }

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
