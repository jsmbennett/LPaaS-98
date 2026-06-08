package catalog

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type AssetEntry struct {
	Filename    string   `json:"filename"`
	Description string   `json:"description"`
	KnownHashes map[string][]string `json:"known_hashes,omitempty"`
	FreeAlts    string   `json:"free_alternative,omitempty"`
}

type GameEntry struct {
	Kind            string       `json:"kind"`
	ID              string       `json:"id"`
	Name            string       `json:"name"`
	Version         string       `json:"version"`
	Description     string       `json:"description"`
	DownloadURL     string       `json:"download_url"`
	SHA256          string       `json:"sha256"`
	MinPlayers      int          `json:"min_players,omitempty"`
	MaxPlayers      int          `json:"max_players,omitempty"`
	NetworkModel    string       `json:"network_model,omitempty"`
	RequiresAssets  []AssetEntry `json:"requires_assets,omitempty"`
}

type Catalog struct {
	SchemaVersion int          `json:"schema_version"`
	Entries       []GameEntry  `json:"entries"`
	FetchedAt     time.Time    `json:"-"`
}

type Client struct {
	catalogURL string
	cacheFile  string
	httpClient *http.Client
}

func New(catalogURL, cacheFile string) *Client {
	return &Client{
		catalogURL: catalogURL,
		cacheFile:  cacheFile,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *Client) Fetch() (*Catalog, error) {
	resp, err := c.httpClient.Get(c.catalogURL)
	if err != nil {
		return nil, fmt.Errorf("fetching catalog: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("catalog fetch failed: %d", resp.StatusCode)
	}

	var cat Catalog
	if err := json.NewDecoder(resp.Body).Decode(&cat); err != nil {
		return nil, fmt.Errorf("parsing catalog: %w", err)
	}

	cat.FetchedAt = time.Now()
	return &cat, nil
}

func (c *Client) Save(cat *Catalog) error {
	data, err := json.MarshalIndent(cat, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling catalog: %w", err)
	}

	if err := os.WriteFile(c.cacheFile, data, 0644); err != nil {
		return fmt.Errorf("writing cache: %w", err)
	}

	return nil
}

func (c *Client) Load() (*Catalog, error) {
	data, err := os.ReadFile(c.cacheFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading cache: %w", err)
	}

	var cat Catalog
	if err := json.Unmarshal(data, &cat); err != nil {
		return nil, fmt.Errorf("parsing cache: %w", err)
	}

	return &cat, nil
}

func (c *Client) FindGame(id string, cat *Catalog) *GameEntry {
	for i := range cat.Entries {
		if cat.Entries[i].ID == id {
			return &cat.Entries[i]
		}
	}
	return nil
}

func (c *Client) DownloadGame(entry *GameEntry, destDir string, onProgress func(int64, int64)) error {
	resp, err := c.httpClient.Get(entry.DownloadURL)
	if err != nil {
		return fmt.Errorf("downloading game: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed: %d", resp.StatusCode)
	}

	if resp.ContentLength == 0 {
		return fmt.Errorf("unknown content length")
	}

	tmpFile := filepath.Join(destDir, ".tmp-download")
	f, err := os.Create(tmpFile)
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	defer f.Close()

	var written int64
	if err := copyWithProgress(f, resp.Body, resp.ContentLength, &written, onProgress); err != nil {
		os.Remove(tmpFile)
		return err
	}

	return nil
}

func copyWithProgress(dst io.Writer, src io.Reader, total int64, written *int64, onProgress func(int64, int64)) error {
	buf := make([]byte, 32*1024)
	for {
		n, err := src.Read(buf)
		if n > 0 {
			if _, writeErr := dst.Write(buf[:n]); writeErr != nil {
				return fmt.Errorf("writing data: %w", writeErr)
			}
			*written += int64(n)
			if onProgress != nil {
				onProgress(*written, total)
			}
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("reading data: %w", err)
		}
	}
	return nil
}
