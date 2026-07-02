package repository

import (
	"path/filepath"
	"sync"
	"testing"
	"time"

	"plugin-execution-system/internal/model"
	"plugin-execution-system/internal/storage"
)

func TestCreateWithIdempotencyConcurrentSameKeyCreatesOneLocalDev(t *testing.T) {
	store, err := storage.OpenJSONStore(filepath.Join(t.TempDir(), "data"))
	if err != nil {
		t.Fatal(err)
	}
	if err := storage.Migrate(store); err != nil {
		t.Fatal(err)
	}
	repo := NewExecutionRepository(store)
	scope := model.NewDefaultScope()
	base := model.Execution{TenantID: scope.TenantID, ProjectID: scope.ProjectID, UserID: "user_demo", PluginIDs: []string{"plugin_echo"}, InputJSON: map[string]any{"x": 1}, InputHash: "ih", PluginIDsHash: "ph", IdempotencyKey: "same-key", Status: model.ExecutionStatusPending, CreatedAt: time.Now().UTC()}
	var wg sync.WaitGroup
	ids := make(chan string, 100)
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			e := base
			e.ID = "exec_concurrent_" + string(rune(i))
			got, _, err := repo.CreateWithIdempotency(e)
			if err != nil {
				t.Errorf("create failed: %v", err)
				return
			}
			ids <- got.ID
		}(i)
	}
	wg.Wait()
	close(ids)
	seen := map[string]bool{}
	for id := range ids {
		seen[id] = true
	}
	if len(seen) != 1 {
		t.Fatalf("expected one idempotent execution, got %v", seen)
	}
	items, err := repo.ListAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("expected exactly one persisted execution, got %d", len(items))
	}
}

func TestListByScopeDoesNotLeakOtherTenant(t *testing.T) {
	store, err := storage.OpenJSONStore(filepath.Join(t.TempDir(), "data"))
	if err != nil {
		t.Fatal(err)
	}
	if err := storage.Migrate(store); err != nil {
		t.Fatal(err)
	}
	repo := NewExecutionRepository(store)
	now := time.Now().UTC()
	_ = repo.Create(model.Execution{TenantID: "tenant_a", ProjectID: "project_a", ID: "exec_a", UserID: "u", Status: model.ExecutionStatusPending, CreatedAt: now})
	_ = repo.Create(model.Execution{TenantID: "tenant_b", ProjectID: "project_b", ID: "exec_b", UserID: "u", Status: model.ExecutionStatusPending, CreatedAt: now})
	items, err := repo.ListByScope(model.ResourceScope{TenantID: "tenant_a", ProjectID: "project_a"})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].ID != "exec_a" {
		t.Fatalf("scope leak or missing result: %+v", items)
	}
}
