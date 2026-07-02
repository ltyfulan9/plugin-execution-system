package artifact

import (
	"context"
	"io"
	"strings"
	"testing"

	"plugin-execution-system/internal/model"
)

func TestLocalStorePutGet(t *testing.T) {
	store, err := NewLocalStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	obj, err := store.Put(context.Background(), model.ResourceScope{TenantID: "t1", ProjectID: "p1"}, ObjectLog, "stdout.log", strings.NewReader("hello"))
	if err != nil {
		t.Fatal(err)
	}
	if obj.SizeBytes != 5 || obj.SHA256 == "" || !strings.Contains(obj.URI, "/t1/p1/log/") {
		t.Fatalf("bad object: %+v", obj)
	}
	r, err := store.Get(context.Background(), obj)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	b, _ := io.ReadAll(r)
	if string(b) != "hello" {
		t.Fatalf("bad body %q", string(b))
	}
}

func TestLocalStoreRejectsBadName(t *testing.T) {
	store, err := NewLocalStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Put(context.Background(), model.NewDefaultScope(), ObjectBlob, "", strings.NewReader("x")); err == nil {
		t.Fatalf("expected bad name")
	}
}
