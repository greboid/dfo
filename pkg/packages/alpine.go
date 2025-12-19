package packages

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"

	"github.com/csmith/apkutils/v2"
	"github.com/csmith/apkutils/v2/keys"
	"gopkg.in/yaml.v3"
)

const (
	apkIndexURLTemplate = "https://dl-cdn.alpinelinux.org/alpine/v%s/%s/x86_64/APKINDEX.tar.gz"
	latestReleaseURL    = "https://dl-cdn.alpinelinux.org/alpine/latest-stable/releases/x86_64/latest-releases.yaml"
)

type AlpineClient struct {
	httpClient    *http.Client
	indexCache    map[string]map[string]*apkutils.PackageInfo
	latestVersion string
	mu            sync.RWMutex
}

func NewAlpineClient() *AlpineClient {
	return &AlpineClient{
		httpClient: &http.Client{},
		indexCache: make(map[string]map[string]*apkutils.PackageInfo),
	}
}

func (c *AlpineClient) FetchIndex(version, repo string) (map[string]*apkutils.PackageInfo, error) {
	cacheKey := fmt.Sprintf("%s:%s", version, repo)

	c.mu.RLock()
	if cached, ok := c.indexCache[cacheKey]; ok {
		c.mu.RUnlock()
		slog.Debug("using cached APKINDEX",
			"version", version,
			"repo", repo,
			"packages", len(cached))
		return cached, nil
	}
	c.mu.RUnlock()

	url := fmt.Sprintf(apkIndexURLTemplate, version, repo)
	slog.Debug("fetching APKINDEX from network",
		"version", version,
		"repo", repo,
		"url", url)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("fetching APKINDEX from %s: %w", url, err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			slog.Error("Unable to fetch latest release", "error", err)
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetching APKINDEX from %s: HTTP %d", url, resp.StatusCode)
	}

	slog.Debug("parsing APKINDEX", "version", version, "repo", repo)
	packages, err := apkutils.ReadApkIndex(resp.Body, keys.X86_64)
	if err != nil {
		return nil, fmt.Errorf("parsing APKINDEX: %w", err)
	}

	slog.Debug("parsed APKINDEX",
		"version", version,
		"repo", repo,
		"packages", len(packages))

	c.mu.Lock()
	c.indexCache[cacheKey] = packages
	c.mu.Unlock()

	return packages, nil
}

func (c *AlpineClient) GetCombinedPackages(version string, repos []string) (map[string]*apkutils.PackageInfo, error) {
	slog.Debug("building combined package map",
		"version", version,
		"repos", repos)

	combined := make(map[string]*apkutils.PackageInfo)

	for _, repo := range repos {
		packages, err := c.FetchIndex(version, repo)
		if err != nil {
			return nil, fmt.Errorf("fetching %s repository: %w", repo, err)
		}

		for name, pkg := range packages {
			combined[name] = pkg
		}
	}

	slog.Debug("combined package map built",
		"version", version,
		"total_packages", len(combined))

	return combined, nil
}

func (c *AlpineClient) GetLatestStableVersion() (string, error) {
	c.mu.RLock()
	if c.latestVersion != "" {
		version := c.latestVersion
		c.mu.RUnlock()
		slog.Debug("using cached latest Alpine version", "version", version)
		return version, nil
	}
	c.mu.RUnlock()

	slog.Debug("fetching latest Alpine version from network")

	resp, err := c.httpClient.Get(latestReleaseURL)
	if err != nil {
		return "", fmt.Errorf("fetching latest release info: %w", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			slog.Error("Unable to fetch latest release", "error", err)
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("fetching latest release info: HTTP %d", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading latest release info: %w", err)
	}

	var releases []struct {
		Version string `yaml:"version"`
		Flavor  string `yaml:"flavor"`
	}

	if err := yaml.Unmarshal(bodyBytes, &releases); err != nil {
		return "", fmt.Errorf("parsing latest release YAML: %w", err)
	}

	if len(releases) == 0 {
		return "", fmt.Errorf("no releases found in latest-releases.yaml")
	}

	version := releases[0].Version
	parts := strings.Split(version, ".")
	if len(parts) < 2 {
		return "", fmt.Errorf("unexpected version format: %s", version)
	}

	majorMinor := fmt.Sprintf("%s.%s", parts[0], parts[1])

	c.mu.Lock()
	c.latestVersion = majorMinor
	c.mu.Unlock()

	slog.Debug("detected latest Alpine version", "version", majorMinor)

	return majorMinor, nil
}
