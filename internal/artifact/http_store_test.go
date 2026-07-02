package artifact

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"plugin-execution-system/internal/model"
)

func TestHTTPObjectStorePutAndGet(t *testing.T) {
	objects := map[string]string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-PES-Artifact-Signature") == "" {
			t.Errorf("expected artifact signature header")
		}
		switch r.Method {
		case http.MethodPut:
			b, _ := io.ReadAll(r.Body)
			objects[r.URL.Path] = string(b)
			w.WriteHeader(http.StatusCreated)
		case http.MethodGet:
			v, ok := objects[r.URL.Path]
			if !ok {
				http.NotFound(w, r)
				return
			}
			_, _ = w.Write([]byte(v))
		default:
			http.Error(w, "bad method", http.StatusMethodNotAllowed)
		}
	}))
	defer server.Close()
	store, err := NewHTTPObjectStore(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	store.HMACSecret = "artifact-secret"
	obj, err := store.Put(context.Background(), model.ResourceScope{TenantID: "t1", ProjectID: "p1"}, ObjectResult, "result.json", strings.NewReader("hello"))
	if err != nil {
		t.Fatal(err)
	}
	if obj.SHA256 == "" || obj.SizeBytes != 5 {
		t.Fatalf("bad object metadata: %#v", obj)
	}
	rc, err := store.Get(context.Background(), obj)
	if err != nil {
		t.Fatal(err)
	}
	defer rc.Close()
	b, _ := io.ReadAll(rc)
	if string(b) != "hello" {
		t.Fatalf("unexpected artifact body: %q", b)
	}
}
