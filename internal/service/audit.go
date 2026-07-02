package service

import (
	"time"

	"plugin-execution-system/internal/model"
	"plugin-execution-system/internal/repository"
)

type AuditService struct{ repo repository.AuditStore }

func NewAuditService(repo repository.AuditStore) *AuditService {
	return &AuditService{repo: repo}
}

func (s *AuditService) Record(userID string, action model.AuditAction, resourceType model.AuditResourceType, resourceID, requestID, message string, detail map[string]any) error {
	return s.RecordScoped(model.NewDefaultScope(), userID, action, resourceType, resourceID, requestID, "", model.AuditDecisionAllow, "", message, detail)
}

func (s *AuditService) RecordScoped(scope model.ResourceScope, actorID string, action model.AuditAction, resourceType model.AuditResourceType, resourceID, requestID, traceID string, decision model.AuditDecision, reason, message string, detail map[string]any) error {
	if s == nil || s.repo == nil {
		return nil
	}
	scope = scope.Normalize()
	if decision == "" {
		decision = model.AuditDecisionAllow
	}
	log := model.AuditLog{
		TenantID:     scope.TenantID,
		ProjectID:    scope.ProjectID,
		ID:           newID("audit"),
		ActorID:      actorID,
		ActorType:    "user",
		UserID:       actorID,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Decision:     decision,
		Reason:       reason,
		RequestID:    requestID,
		TraceID:      traceID,
		Message:      message,
		DetailJSON:   detail,
		CreatedAt:    time.Now().UTC(),
	}
	if detail != nil {
		if v, _ := detail["plugin_digest"].(string); v != "" {
			log.PluginDigest = v
		}
		if v, _ := detail["input_hash"].(string); v != "" {
			log.InputHash = v
		}
		if v, _ := detail["result_hash"].(string); v != "" {
			log.ResultHash = v
		}
	}
	return s.repo.Create(log)
}

func (s *AuditService) ListAuditLogs() ([]model.AuditLog, error) { return s.repo.List() }
func (s *AuditService) ListByResource(t model.AuditResourceType, id string) ([]model.AuditLog, error) {
	return s.repo.ListByResource(t, id)
}
