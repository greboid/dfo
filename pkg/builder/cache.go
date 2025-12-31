package builder

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/greboid/dfo/pkg/util"
)

type BuildCache struct {
	cachePath string
	entries   map[string]*CacheEntry
	fs        util.WritableFS
}

type CacheEntry struct {
	ContainerName string    `json:"container_name"`
	InputHash     string    `json:"input_hash"`
	BuildDigest   string    `json:"build_digest"`
	Timestamp     time.Time `json:"timestamp"`
	ConfigPath    string    `json:"config_path"`
}

type CacheManifest struct {
	Version int                    `json:"version"`
	Entries map[string]*CacheEntry `json:"entries"`
}

func NewBuildCache(baseDir string, fs util.WritableFS) (*BuildCache, error) {
	cachePath := filepath.Join(baseDir, ".dfo-build-cache.json")

	cache := &BuildCache{
		cachePath: cachePath,
		entries:   make(map[string]*CacheEntry),
		fs:        fs,
	}

	if err := cache.load(); err != nil {
		return cache, nil
	}

	return cache, nil
}

func (c *BuildCache) load() error {
	data, err := c.fs.ReadFile(c.cachePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("reading cache file: %w", err)
	}

	var manifest CacheManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return fmt.Errorf("parsing cache file: %w", err)
	}

	c.entries = manifest.Entries
	return nil
}

func (c *BuildCache) Save() error {
	manifest := CacheManifest{
		Version: 1,
		Entries: c.entries,
	}

	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling cache: %w", err)
	}

	if err := c.fs.WriteFile(c.cachePath, data, 0644); err != nil {
		return fmt.Errorf("writing cache file: %w", err)
	}

	return nil
}

func (c *BuildCache) HashConfigFile(configPath string) (string, error) {
	data, err := c.fs.ReadFile(configPath)
	if err != nil {
		return "", fmt.Errorf("reading config file: %w", err)
	}

	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:]), nil
}

func (c *BuildCache) NeedsRebuild(containerName, configPath string) (bool, error) {
	currentHash, err := c.HashConfigFile(configPath)
	if err != nil {
		return true, fmt.Errorf("hashing config: %w", err)
	}

	entry, exists := c.entries[containerName]
	if !exists {
		return true, nil
	}

	if entry.InputHash != currentHash {
		return true, nil
	}

	return false, nil
}

func (c *BuildCache) Record(result *BuildResult, configPath string) error {
	inputHash, err := c.HashConfigFile(configPath)
	if err != nil {
		return fmt.Errorf("hashing config: %w", err)
	}

	c.entries[result.ContainerName] = &CacheEntry{
		ContainerName: result.ContainerName,
		InputHash:     inputHash,
		BuildDigest:   result.Digest,
		Timestamp:     time.Now(),
		ConfigPath:    configPath,
	}

	return nil
}

func (c *BuildCache) GetCachedDigest(containerName string) (string, bool) {
	entry, exists := c.entries[containerName]
	if !exists {
		return "", false
	}
	return entry.BuildDigest, true
}

func (c *BuildCache) Invalidate(containerName string) {
	delete(c.entries, containerName)
}

func (c *BuildCache) InvalidateAll() {
	c.entries = make(map[string]*CacheEntry)
}
