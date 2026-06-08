package registry

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type AssetRequirement struct {
	Filename    string                 `json:"filename"`
	Description string                 `json:"description"`
	KnownHashes map[string][]string    `json:"known_hashes,omitempty"`
	FreeAlts    string                 `json:"free_alternative,omitempty"`
}

type Game struct {
	ID               string              `json:"id"`
	Name             string              `json:"name"`
	Version          string              `json:"version"`
	Description      string              `json:"description"`
	MinPlayers       int                 `json:"min_players"`
	MaxPlayers       int                 `json:"max_players"`
	NetworkModel     string              `json:"network_model"`
	WASM             string              `json:"wasm"`
	JSLoader         string              `json:"js_loader"`
	RequiresAssets   []AssetRequirement  `json:"requires_assets"`
	Icons            map[string]string   `json:"icons,omitempty"`
	ShortcutName     string              `json:"shortcut_name,omitempty"`
	License          string              `json:"license,omitempty"`
	SourceURL        string              `json:"source_url,omitempty"`
}

type GameEntry struct {
	Game        Game   `json:"game"`
	AssetStatus string `json:"asset_status"`
}

type Registry struct {
	games map[string]*GameEntry
}

func New() *Registry {
	return &Registry{
		games: make(map[string]*GameEntry),
	}
}

func (r *Registry) Scan(gamesDir string) error {
	entries, err := os.ReadDir(gamesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("scanning games dir: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		manifestPath := filepath.Join(gamesDir, entry.Name(), "manifest.json")
		data, err := os.ReadFile(manifestPath)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			fmt.Fprintf(os.Stderr, "warning: reading manifest %s: %v\n", manifestPath, err)
			continue
		}

		var game Game
		if err := json.Unmarshal(data, &game); err != nil {
			fmt.Fprintf(os.Stderr, "warning: parsing manifest %s: %v\n", manifestPath, err)
			continue
		}

		status := determineAssetStatus(&game)
		r.games[game.ID] = &GameEntry{
			Game:        game,
			AssetStatus: status,
		}
	}

	return nil
}

func (r *Registry) List() []*GameEntry {
	var result []*GameEntry
	for _, entry := range r.games {
		result = append(result, entry)
	}
	return result
}

func (r *Registry) Get(id string) *GameEntry {
	return r.games[id]
}

func determineAssetStatus(game *Game) string {
	if len(game.RequiresAssets) == 0 {
		return "not_required"
	}
	return "missing"
}
