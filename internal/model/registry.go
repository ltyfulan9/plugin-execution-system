package model

import "time"

type PluginRegistryRecord struct {
	ID           string    `json:"id"`
	PluginID     string    `json:"plugin_id"`
	Name         string    `json:"name"`
	Version      string    `json:"version"`
	SourcePath   string    `json:"source_path"`
	ManifestHash string    `json:"manifest_hash"`
	LastSeenAt   time.Time `json:"last_seen_at"`
	SyncedAt     time.Time `json:"synced_at"`
}

type RegistrySyncResult struct {
	Created int      `json:"created"`
	Updated int      `json:"updated"`
	Removed int      `json:"removed"`
	Errors  []string `json:"errors"`
}

func (r PluginRegistryRecord) RegistryKey() string { return r.Name + "@" + r.Version }
