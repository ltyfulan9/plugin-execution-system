package repository

import (
	"plugin-execution-system/internal/model"
	"plugin-execution-system/internal/storage"
)

type AuthStore interface {
	CreateUser(model.User) error
	GetUserByID(string) (model.User, bool, error)
	GetUserByTokenHash(string) (model.User, bool, error)
	GetUserByUsername(string) (model.User, bool, error)
	ListUsers() ([]model.User, error)
}

type PluginStore interface {
	Create(model.Plugin) error
	Update(model.Plugin) error
	GetByID(string) (model.Plugin, bool, error)
	GetByNameVersion(string, string) (model.Plugin, bool, error)
	List() ([]model.Plugin, error)
	UpdateStatus(string, model.PluginStatus) error
	UpdateError(string, string) error
	MarkRemoved(string) error
}

type RegistryStore interface {
	UpsertRegistryRecord(model.PluginRegistryRecord) error
	GetByPluginID(string) (model.PluginRegistryRecord, bool, error)
	ListRegistryRecords() ([]model.PluginRegistryRecord, error)
}

type ExecutionStore interface {
	Create(model.Execution) error
	CreateWithIdempotency(model.Execution) (model.Execution, bool, error)
	Update(model.Execution) error
	GetByID(string) (model.Execution, bool, error)
	ListByUserID(string) ([]model.Execution, error)
	ListAll() ([]model.Execution, error)
	ListByScope(model.ResourceScope) ([]model.Execution, error)
	UpdateStatus(string, model.ExecutionStatus, string) error
	FindByIdempotencyKey(string, string) (model.Execution, bool, error)
	ListByStatuses(...model.ExecutionStatus) ([]model.Execution, error)
}

type ResultStore interface {
	Create(model.ExecutionResult) error
	BatchCreate([]model.ExecutionResult) error
	GetByExecutionID(string) ([]model.ExecutionResult, error)
	DeleteByExecutionID(string) error
}

type AuditStore interface {
	Create(model.AuditLog) error
	List() ([]model.AuditLog, error)
	ListByResource(model.AuditResourceType, string) ([]model.AuditLog, error)
	ListByRequestID(string) ([]model.AuditLog, error)
}

type ExecutionEventStore interface {
	Create(model.ExecutionEvent) error
	ListByExecutionID(string) ([]model.ExecutionEvent, error)
}

type ExecutionAttemptStore interface {
	Create(model.ExecutionAttempt) error
	Update(model.ExecutionAttempt) error
	ListByExecutionID(string) ([]model.ExecutionAttempt, error)
	NextAttemptNo(string) (int, error)
}

type WebhookStore interface {
	CreateEndpoint(model.WebhookEndpoint) error
	UpdateEndpoint(model.WebhookEndpoint) error
	GetEndpointByID(string) (model.WebhookEndpoint, bool, error)
	ListEndpoints() ([]model.WebhookEndpoint, error)
	ListEnabledEndpoints() ([]model.WebhookEndpoint, error)
	DeleteEndpoint(string) error
	CreateDelivery(model.WebhookDelivery) error
	UpdateDelivery(model.WebhookDelivery) error
	ListDeliveries(string) ([]model.WebhookDelivery, error)
}

type Repositories struct {
	Auth      AuthStore
	Plugin    PluginStore
	Registry  RegistryStore
	Execution ExecutionStore
	Result    ResultStore
	Audit     AuditStore
	Event     ExecutionEventStore
	Attempt   ExecutionAttemptStore
	Webhook   WebhookStore
}

func NewRepositories(store *storage.JSONStore) *Repositories {
	return &Repositories{
		Auth: NewAuthRepository(store), Plugin: NewPluginRepository(store), Registry: NewRegistryRepository(store),
		Execution: NewExecutionRepository(store), Result: NewResultRepository(store), Audit: NewAuditRepository(store),
		Event: NewExecutionEventRepository(store), Attempt: NewExecutionAttemptRepository(store), Webhook: NewWebhookRepository(store),
	}
}
