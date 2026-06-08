package registry

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScanEmptyDir(t *testing.T) {
	tmpDir := t.TempDir()
	reg := New()
	err := reg.Scan(tmpDir)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(reg.List()) != 0 {
		t.Fatalf("expected 0 games, got %d", len(reg.List()))
	}
}

func TestScanGameWithNoAssets(t *testing.T) {
	tmpDir := t.TempDir()
	gameDir := filepath.Join(tmpDir, "test-game")
	os.Mkdir(gameDir, 0755)

	manifest := `{
		"id": "test-game",
		"name": "Test Game",
		"version": "1.0.0",
		"description": "A test game",
		"min_players": 2,
		"max_players": 4,
		"network_model": "host-client",
		"wasm": "game.wasm",
		"js_loader": "loader.js",
		"requires_assets": []
	}`

	manifestPath := filepath.Join(gameDir, "manifest.json")
	os.WriteFile(manifestPath, []byte(manifest), 0644)

	reg := New()
	reg.Scan(tmpDir)

	games := reg.List()
	if len(games) != 1 {
		t.Fatalf("expected 1 game, got %d", len(games))
	}

	if games[0].AssetStatus != "not_required" {
		t.Fatalf("expected asset_status 'not_required', got '%s'", games[0].AssetStatus)
	}
}

func TestScanGameWithAssets(t *testing.T) {
	tmpDir := t.TempDir()
	gameDir := filepath.Join(tmpDir, "test-game")
	os.Mkdir(gameDir, 0755)

	manifest := `{
		"id": "test-game",
		"name": "Test Game",
		"version": "1.0.0",
		"description": "A test game",
		"min_players": 2,
		"max_players": 4,
		"network_model": "host-client",
		"wasm": "game.wasm",
		"js_loader": "loader.js",
		"requires_assets": [
			{
				"filename": "game.wad",
				"description": "Game WAD",
				"free_alternative": "freegame"
			}
		]
	}`

	manifestPath := filepath.Join(gameDir, "manifest.json")
	os.WriteFile(manifestPath, []byte(manifest), 0644)

	reg := New()
	reg.Scan(tmpDir)

	games := reg.List()
	if len(games) != 1 {
		t.Fatalf("expected 1 game, got %d", len(games))
	}

	if games[0].AssetStatus != "missing" {
		t.Fatalf("expected asset_status 'missing', got '%s'", games[0].AssetStatus)
	}
}
