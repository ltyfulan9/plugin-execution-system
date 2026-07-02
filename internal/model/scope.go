package model

const (
	DefaultTenantID  = "tenant_default"
	DefaultProjectID = "project_default"
)

type ResourceScope struct {
	TenantID  string `json:"tenant_id"`
	ProjectID string `json:"project_id"`
}

func NewDefaultScope() ResourceScope {
	return ResourceScope{TenantID: DefaultTenantID, ProjectID: DefaultProjectID}
}

func (s ResourceScope) IsZero() bool { return s.TenantID == "" || s.ProjectID == "" }

func (s ResourceScope) Normalize() ResourceScope {
	if s.TenantID == "" {
		s.TenantID = DefaultTenantID
	}
	if s.ProjectID == "" {
		s.ProjectID = DefaultProjectID
	}
	return s
}

func SameScope(a, b ResourceScope) bool {
	a = a.Normalize()
	b = b.Normalize()
	return a.TenantID == b.TenantID && a.ProjectID == b.ProjectID
}
