package util

import (
	"errors"
	"io/fs"
	"path"
	"sort"
	"strings"
	"testing/fstest"
	"time"
)

type TestFS struct {
	MapFS fstest.MapFS
}

func NewTestFS() *TestFS {
	return &TestFS{
		MapFS: make(fstest.MapFS),
	}
}

func (t *TestFS) Open(name string) (fs.File, error) {
	return t.MapFS.Open(name)
}

func (t *TestFS) Stat(name string) (fs.FileInfo, error) {
	return fs.Stat(t.MapFS, name)
}

func (t *TestFS) ReadFile(name string) ([]byte, error) {
	return fs.ReadFile(t.MapFS, name)
}

func (t *TestFS) WriteFile(name string, data []byte, perm fs.FileMode) error {
	dir := path.Dir(name)
	if dir != "." && dir != "" {
		t.ensureDir(dir)
	}

	t.MapFS[name] = &fstest.MapFile{
		Data:    data,
		Mode:    perm,
		ModTime: time.Now(),
	}
	return nil
}

func (t *TestFS) MkdirAll(p string, _ fs.FileMode) error {
	t.ensureDir(p)
	return nil
}

func (t *TestFS) ensureDir(p string) {
	parts := strings.Split(p, "/")
	current := ""
	for _, part := range parts {
		if part == "" || part == "." {
			continue
		}
		if current == "" {
			current = part
		} else {
			current = current + "/" + part
		}
		if _, exists := t.MapFS[current]; !exists {
			t.MapFS[current] = &fstest.MapFile{
				Mode:    fs.ModeDir | 0755,
				ModTime: time.Now(),
			}
		}
	}
}

func (t *TestFS) WalkDir(root string, fn fs.WalkDirFunc) error {
	var paths []string
	for p := range t.MapFS {
		paths = append(paths, p)
	}
	sort.Strings(paths)

	var skippedDirs []string

	isUnderSkipped := func(p string) bool {
		for _, skipped := range skippedDirs {
			if strings.HasPrefix(p, skipped+"/") {
				return true
			}
		}
		return false
	}

	if root == "." || root == "" {
		rootInfo := &mapDirEntry{name: ".", isDir: true}
		if err := fn(".", rootInfo, nil); err != nil {
			if errors.Is(err, fs.SkipDir) {
				return nil
			}
			return err
		}

		for _, p := range paths {
			if isUnderSkipped(p) {
				continue
			}
			if stop, skippedDir, err := t.visitPath(p, fn); err != nil {
				return err
			} else if stop && skippedDir != "" {
				skippedDirs = append(skippedDirs, skippedDir)
			}
		}
		return nil
	}

	rootInfo, err := t.Stat(root)
	if err != nil {
		return fn(root, nil, err)
	}

	rootEntry := &mapDirEntry{
		name:  path.Base(root),
		isDir: rootInfo.IsDir(),
		info:  rootInfo,
	}
	if err := fn(root, rootEntry, nil); err != nil {
		if errors.Is(err, fs.SkipDir) {
			return nil
		}
		return err
	}

	prefix := root + "/"
	for _, p := range paths {
		if !strings.HasPrefix(p, prefix) && p != root {
			continue
		}
		if p == root {
			continue
		}
		if isUnderSkipped(p) {
			continue
		}

		if stop, skippedDir, err := t.visitPath(p, fn); err != nil {
			return err
		} else if stop && skippedDir != "" {
			skippedDirs = append(skippedDirs, skippedDir)
		}
	}

	return nil
}

func (t *TestFS) visitPath(p string, fn fs.WalkDirFunc) (skip bool, skippedDir string, err error) {
	info, statErr := t.Stat(p)
	if statErr != nil {
		if err := fn(p, nil, statErr); err != nil && !errors.Is(err, fs.SkipDir) {
			return false, "", err
		}
		return true, "", nil
	}
	entry := &mapDirEntry{
		name:  path.Base(p),
		isDir: info.IsDir(),
		info:  info,
	}
	if err := fn(p, entry, nil); err != nil {
		if errors.Is(err, fs.SkipDir) {
			if info.IsDir() {
				return true, p, nil
			}
			return true, "", nil
		}
		return false, "", err
	}
	return false, "", nil
}

type mapDirEntry struct {
	name  string
	isDir bool
	info  fs.FileInfo
}

func (e *mapDirEntry) Name() string {
	return e.name
}

func (e *mapDirEntry) IsDir() bool {
	return e.isDir
}

func (e *mapDirEntry) Type() fs.FileMode {
	if e.isDir {
		return fs.ModeDir
	}
	return 0
}

func (e *mapDirEntry) Info() (fs.FileInfo, error) {
	return e.info, nil
}
