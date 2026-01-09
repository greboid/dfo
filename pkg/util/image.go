package util

import (
	"fmt"
	"strings"
)

func FormatImageRefFromName(registry, name string) string {
	if registry != "" {
		return fmt.Sprintf("%s/%s:latest", registry, name)
	}
	return fmt.Sprintf("%s:latest", name)
}

func NormalizeDigest(digest string) string {
	if digest == "" || digest == "<none>" {
		return digest
	}
	if !strings.HasPrefix(digest, "sha256:") {
		return "sha256:" + digest
	}
	return digest
}

func FormatFullRef(imageName, digest string) string {
	return fmt.Sprintf("%s@%s", imageName, digest)
}
