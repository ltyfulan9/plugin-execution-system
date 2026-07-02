package storage

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
)

type JSONStore struct {
	dir string
	mu  sync.RWMutex
}

func OpenJSONStore(dir string) (*JSONStore, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return &JSONStore{dir: dir}, nil
}

func (s *JSONStore) Path(name string) string { return filepath.Join(s.dir, name+".json") }

func (s *JSONStore) Load(name string, dest any) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	b, err := os.ReadFile(s.Path(name))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if len(b) == 0 {
		return nil
	}
	return json.Unmarshal(b, dest)
}

func (s *JSONStore) Save(name string, value any) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.Path(name) + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, s.Path(name))
}

func (s *JSONStore) EnsureArrayFiles(names ...string) error {
	for _, name := range names {
		path := s.Path(name)
		if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
			if err := os.WriteFile(path, []byte("[]"), 0o644); err != nil {
				return err
			}
		} else if err != nil {
			return err
		}
	}
	return nil
}
