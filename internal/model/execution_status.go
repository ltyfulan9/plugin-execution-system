package model

type ExecutionStatus string

const (
	ExecutionStatusPending        ExecutionStatus = "Pending"
	ExecutionStatusQueued         ExecutionStatus = "Queued"
	ExecutionStatusRunning        ExecutionStatus = "Running"
	ExecutionStatusSuccess        ExecutionStatus = "Success"
	ExecutionStatusPartialSuccess ExecutionStatus = "PartialSuccess"
	ExecutionStatusFailed         ExecutionStatus = "Failed"
	ExecutionStatusTimeout        ExecutionStatus = "Timeout"
	ExecutionStatusCanceled       ExecutionStatus = "Canceled"
)

func IsValidExecutionStatus(s ExecutionStatus) bool {
	switch s {
	case ExecutionStatusPending, ExecutionStatusQueued, ExecutionStatusRunning, ExecutionStatusSuccess, ExecutionStatusPartialSuccess, ExecutionStatusFailed, ExecutionStatusTimeout, ExecutionStatusCanceled:
		return true
	default:
		return false
	}
}

func IsFinalExecutionStatus(s ExecutionStatus) bool {
	switch s {
	case ExecutionStatusSuccess, ExecutionStatusPartialSuccess, ExecutionStatusFailed, ExecutionStatusTimeout, ExecutionStatusCanceled:
		return true
	default:
		return false
	}
}
