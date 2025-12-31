package builder

import (
	"maps"
	"sync"
)

type BuildResult struct {
	ContainerName string
	ImageName     string
	Digest        string
	FullRef       string
	Size          int64
}

type BuildRegistry struct {
	mu     sync.RWMutex
	builds map[string]*BuildResult
}

func NewBuildRegistry() *BuildRegistry {
	return &BuildRegistry{
		builds: make(map[string]*BuildResult),
	}
}

func (r *BuildRegistry) Record(result *BuildResult) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.builds[result.ContainerName] = result
}

func (r *BuildRegistry) Get(containerName string) (*BuildResult, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result, ok := r.builds[containerName]
	return result, ok
}

func (r *BuildRegistry) GetAll() map[string]*BuildResult {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return maps.Clone(r.builds)
}

func (r *BuildRegistry) Has(containerName string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.builds[containerName]
	return ok
}
