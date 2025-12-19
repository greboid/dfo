package versions

import "strings"

type VersionSpec struct {
	Key   string
	Value string
}

type VersionMetadata struct {
	Version  string
	URL      string
	Checksum string
}

func (s VersionSpec) IsLatest() bool {
	return strings.HasPrefix(s.Value, "latest")
}

func (s VersionSpec) IsGitRepo() bool {
	return strings.HasPrefix(s.Key, "http://") || strings.HasPrefix(s.Key, "https://")
}

func (s VersionSpec) VersionType() string {
	if s.IsGitRepo() {
		return "git"
	}

	keyLower := strings.ToLower(s.Key)
	switch {
	case keyLower == "go" || keyLower == "golang":
		return "go"
	case keyLower == "postgres" || keyLower == "postgresql" || strings.HasPrefix(keyLower, "postgres"):
		return "postgres"
	case keyLower == "alpine":
		return "alpine"
	default:
		return "unknown"
	}
}
