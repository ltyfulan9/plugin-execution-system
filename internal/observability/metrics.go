package observability

import (
	"expvar"
	"strconv"
	"strings"

	"plugin-execution-system/internal/model"
)

var (
	HTTPRequestsTotal        = expvar.NewMap("pes_http_requests_total")
	HTTPStatusTotal          = expvar.NewMap("pes_http_status_total")
	TaskSubmittedTotal       = expvar.NewInt("pes_task_submitted_total")
	TaskRecoveredTotal       = expvar.NewInt("pes_task_recovered_total")
	TaskCompletedTotal       = expvar.NewMap("pes_task_completed_total")
	QueueFullTotal           = expvar.NewInt("pes_queue_full_total")
	QueueLeaseTotal          = expvar.NewInt("pes_queue_lease_total")
	QueueAckTotal            = expvar.NewInt("pes_queue_ack_total")
	QueueNackTotal           = expvar.NewInt("pes_queue_nack_total")
	QueueReclaimedTotal      = expvar.NewInt("pes_queue_reclaimed_total")
	WorkerHeartbeatTotal     = expvar.NewInt("pes_worker_heartbeat_total")
	WorkerHandlerErrorTotal  = expvar.NewInt("pes_worker_handler_error_total")
	IdempotencyHitsTotal     = expvar.NewInt("pes_idempotency_hits_total")
	IdempotencyConflictTotal = expvar.NewInt("pes_idempotency_conflict_total")
	PluginStartedTotal       = expvar.NewMap("pes_plugin_started_total")
	PluginCompletedTotal     = expvar.NewMap("pes_plugin_completed_total")
	SandboxDenialsTotal      = expvar.NewInt("pes_sandbox_denials_total")
	WebhookRetryTotal        = expvar.NewInt("pes_webhook_retry_total")
	WebhookRetryErrorTotal   = expvar.NewInt("pes_webhook_retry_error_total")
)

func IncHTTP(method, path string, status int) {
	HTTPRequestsTotal.Add(method+" "+path, 1)
	HTTPStatusTotal.Add(strconv.Itoa(status), 1)
}

func IncTaskSubmitted()       { TaskSubmittedTotal.Add(1) }
func IncTaskRecovered()       { TaskRecoveredTotal.Add(1) }
func IncQueueFull()           { QueueFullTotal.Add(1) }
func IncQueueLease(n int)     { QueueLeaseTotal.Add(int64(n)) }
func IncQueueAck()            { QueueAckTotal.Add(1) }
func IncQueueNack()           { QueueNackTotal.Add(1) }
func IncQueueReclaimed(n int) { QueueReclaimedTotal.Add(int64(n)) }
func IncWorkerHeartbeat()     { WorkerHeartbeatTotal.Add(1) }
func IncWorkerHandlerError()  { WorkerHandlerErrorTotal.Add(1) }
func IncIdempotencyHit()      { IdempotencyHitsTotal.Add(1) }
func IncIdempotencyConflict() { IdempotencyConflictTotal.Add(1) }
func IncSandboxDenial()       { SandboxDenialsTotal.Add(1) }
func IncWebhookRetry(n int)   { WebhookRetryTotal.Add(int64(n)) }
func IncWebhookRetryError()   { WebhookRetryErrorTotal.Add(1) }

func IncTaskCompleted(status model.ExecutionStatus) {
	TaskCompletedTotal.Add(string(status), 1)
}

func IncPluginStarted(name string) {
	PluginStartedTotal.Add(name, 1)
}

func IncPluginCompleted(status model.PluginResultStatus) {
	PluginCompletedTotal.Add(string(status), 1)
}

func PrometheusText() string {
	var b strings.Builder
	expvar.Do(func(kv expvar.KeyValue) {
		name := strings.ReplaceAll(kv.Key, "pes_", "pes_")
		switch v := kv.Value.(type) {
		case *expvar.Int:
			b.WriteString("# TYPE " + name + " counter\n")
			b.WriteString(name + " " + v.String() + "\n")
		case *expvar.Map:
			v.Do(func(item expvar.KeyValue) {
				b.WriteString(name + "{key=\"" + escapeLabel(item.Key) + "\"} " + item.Value.String() + "\n")
			})
		}
	})
	return b.String()
}

func escapeLabel(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	return s
}
