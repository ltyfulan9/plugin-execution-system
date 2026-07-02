package storage

func Migrate(store *JSONStore) error {
	return store.EnsureArrayFiles("users", "plugins", "plugin_registry", "executions", "execution_results", "execution_events", "execution_attempts", "audit_logs", "webhooks", "webhook_deliveries")
}
