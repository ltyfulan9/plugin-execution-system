package policy

import (
	"context"
	"testing"

	"plugin-execution-system/internal/model"
)

func TestRBACAllowsUserCreateInSameScope(t *testing.T) {
	eng := NewRBACEngine()
	dec, err := eng.Evaluate(context.Background(), Input{Subject: model.PolicySubject{TenantID: "t1", ProjectID: "p1", ActorID: "u1", Role: model.RoleUser}, Action: ActionExecutionCreate, Resource: model.PolicyResource{TenantID: "t1", ProjectID: "p1", Type: "project", ID: "p1"}})
	if err != nil || dec.Decision != model.PolicyDecisionAllow {
		t.Fatalf("expected allow, got %+v err=%v", dec, err)
	}
}

func TestRBACDeniesCrossScope(t *testing.T) {
	eng := NewRBACEngine()
	dec, err := eng.Evaluate(context.Background(), Input{Subject: model.PolicySubject{TenantID: "t1", ProjectID: "p1", ActorID: "u1", Role: model.RoleAdmin}, Action: ActionAuditRead, Resource: model.PolicyResource{TenantID: "t2", ProjectID: "p1", Type: "audit", ID: "a1"}})
	if err != nil || dec.Decision != model.PolicyDecisionDeny {
		t.Fatalf("expected deny, got %+v err=%v", dec, err)
	}
}

func TestRBACDeniesUserPluginManage(t *testing.T) {
	eng := NewRBACEngine()
	dec, err := eng.Evaluate(context.Background(), Input{Subject: model.PolicySubject{TenantID: "t1", ProjectID: "p1", ActorID: "u1", Role: model.RoleUser}, Action: ActionPluginManage, Resource: model.PolicyResource{TenantID: "t1", ProjectID: "p1", Type: "plugin", ID: "p1"}})
	if err != nil || dec.Decision != model.PolicyDecisionDeny {
		t.Fatalf("expected deny, got %+v err=%v", dec, err)
	}
}
