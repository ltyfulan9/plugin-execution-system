package response

const (
	CodeOK                    = "OK"
	CodeInvalidArgument       = "INVALID_ARGUMENT"
	CodeUnauthorized          = "UNAUTHORIZED"
	CodeForbidden             = "FORBIDDEN"
	CodeNotFound              = "NOT_FOUND"
	CodePluginNotFound        = "PLUGIN_NOT_FOUND"
	CodePluginDisabled        = "PLUGIN_DISABLED"
	CodePluginStateInvalid    = "PLUGIN_STATE_INVALID"
	CodeManifestInvalid       = "MANIFEST_INVALID"
	CodeExecutionNotFound     = "EXECUTION_NOT_FOUND"
	CodeExecutionStateInvalid = "EXECUTION_STATE_INVALID"
	CodeIdempotencyConflict   = "IDEMPOTENCY_CONFLICT"
	CodeQueueFull             = "QUEUE_FULL"
	CodePluginRuntimeError    = "PLUGIN_RUNTIME_ERROR"
	CodePluginTimeout         = "PLUGIN_TIMEOUT"
	CodePluginInvalidOutput   = "PLUGIN_INVALID_OUTPUT"
	CodeStorageError          = "STORAGE_ERROR"
	CodeInternalError         = "INTERNAL_ERROR"
)
