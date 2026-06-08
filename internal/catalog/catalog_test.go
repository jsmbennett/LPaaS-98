package catalog

import (
	"encoding/json"
	"path/filepath"
	"testing"
)

func TestLoadSaveCatalog(t *testing.T) {
	tmpDir := t.TempDir()
	cacheFile := filepath.Join(tmpDir, "catalog.json")

	cat := &Catalog{
		SchemaVersion: 1,
		Entries: []GameEntry{
			{
				Kind:    "game",
				ID:      "test",
				Name:    "Test Game",
				Version: "1.0.0",
			},
		},
	}

	client := New("https://example.com/catalog.json", cacheFile)
	if err := client.Save(cat); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	loaded, err := client.Load()
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}

	if loaded == nil {
		t.Fatal("loaded catalog is nil")
	}

	if len(loaded.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(loaded.Entries))
	}

	if loaded.Entries[0].ID != "test" {
		t.Fatalf("expected ID 'test', got '%s'", loaded.Entries[0].ID)
	}
}

func TestLoadNonexistentCatalog(t *testing.T) {
	tmpDir := t.TempDir()
	cacheFile := filepath.Join(tmpDir, "nonexistent.json")

	client := New("https://example.com/catalog.json", cacheFile)
	loaded, err := client.Load()
	if err != nil {
		t.Fatalf("load should not error on missing file: %v", err)
	}

	if loaded != nil {
		t.Fatal("loaded catalog should be nil for missing file")
	}
}

func TestFindGame(t *testing.T) {
	cat := &Catalog{
		Entries: []GameEntry{
			{ID: "game1", Name: "Game 1"},
			{ID: "game2", Name: "Game 2"},
		},
	}

	client := New("", "")
	found := client.FindGame("game1", cat)
	if found == nil {
		t.Fatal("game not found")
	}

	if found.ID != "game1" {
		t.Fatalf("wrong game found: %s", found.ID)
	}

	notFound := client.FindGame("nonexistent", cat)
	if notFound != nil {
		t.Fatal("expected nil for nonexistent game")
	}
}

func TestMarshalCatalog(t *testing.T) {
	cat := &Catalog{
		SchemaVersion: 1,
		Entries: []GameEntry{
			{
				Kind:       "game",
				ID:         "test",
				Name:       "Test",
				DownloadURL: "https://example.com/test.tar.gz",
				SHA256:      "abc123",
			},
		},
	}

	data, err := json.MarshalIndent(cat, "", "  ")
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	if len(data) == 0 {
		t.Fatal("marshaled data is empty")
	}

	var unmarshaled Catalog
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if unmarshaled.SchemaVersion != 1 {
		t.Fatalf("schema version mismatch: %d", unmarshaled.SchemaVersion)
	}
}
