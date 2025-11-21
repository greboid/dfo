package util

import (
	"io/fs"
	"os"
	"path/filepath"
)

type WritableFS interface {
	fs.FS
	fs.StatFS
	fs.ReadFileFS

	WriteFile(name string, data []byte, perm fs.FileMode) error

	MkdirAll(path string, perm fs.FileMode) error
}

type WalkableFS interface {
	WritableFS

	WalkDir(root string, fn fs.WalkDirFunc) error
}

type OSFS struct{}

func (OSFS) Open(name string) (fs.File, error) {
	return os.Open(name)
}

func (OSFS) Stat(name string) (fs.FileInfo, error) {
	return os.Stat(name)
}

func (OSFS) ReadFile(name string) ([]byte, error) {
	return os.ReadFile(name)
}

func (OSFS) WriteFile(name string, data []byte, perm fs.FileMode) error {
	return os.WriteFile(name, data, perm)
}

func (OSFS) MkdirAll(path string, perm fs.FileMode) error {
	return os.MkdirAll(path, perm)
}

func (OSFS) WalkDir(root string, fn fs.WalkDirFunc) error {
	return filepath.WalkDir(root, fn)
}

func DefaultFS() WalkableFS {
	return OSFS{}
}
