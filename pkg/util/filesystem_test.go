package util

import (
	iofs "io/fs"
	"os"
	"path/filepath"
	"testing"
)

func TestOSFS_Open(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "osfs_test_*.txt")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer func(name string) {
		_ = os.Remove(name)
	}(tmpFile.Name())
	_ = tmpFile.Close()

	osfs := OSFS{}

	t.Run("open existing file", func(t *testing.T) {
		f, err := osfs.Open(tmpFile.Name())
		if err != nil {
			t.Errorf("Open() error = %v, want nil", err)
		}
		if f == nil {
			t.Error("Open() returned nil file")
		}
		if f != nil {
			_ = f.Close()
		}
	})

	t.Run("open non-existent file", func(t *testing.T) {
		_, err := osfs.Open("/non/existent/file")
		if err == nil {
			t.Error("Open() error = nil, want error for non-existent file")
		}
	})
}

func TestOSFS_Stat(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "osfs_stat_*.txt")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer func(name string) {
		_ = os.Remove(name)
	}(tmpFile.Name())
	_ = tmpFile.Close()

	osfs := OSFS{}

	t.Run("stat existing file", func(t *testing.T) {
		info, err := osfs.Stat(tmpFile.Name())
		if err != nil {
			t.Errorf("Stat() error = %v, want nil", err)
		}
		if info == nil {
			t.Error("Stat() returned nil FileInfo")
		}
	})

	t.Run("stat non-existent file", func(t *testing.T) {
		_, err := osfs.Stat("/non/existent/file")
		if err == nil {
			t.Error("Stat() error = nil, want error for non-existent file")
		}
	})
}

func TestOSFS_ReadFile(t *testing.T) {
	content := []byte("test content")
	tmpFile, err := os.CreateTemp("", "osfs_read_*.txt")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer func(name string) {
		_ = os.Remove(name)
	}(tmpFile.Name())

	if _, err := tmpFile.Write(content); err != nil {
		t.Fatalf("failed to write to temp file: %v", err)
	}
	_ = tmpFile.Close()

	osfs := OSFS{}

	t.Run("read existing file", func(t *testing.T) {
		data, err := osfs.ReadFile(tmpFile.Name())
		if err != nil {
			t.Errorf("ReadFile() error = %v, want nil", err)
		}
		if string(data) != string(content) {
			t.Errorf("ReadFile() = %q, want %q", data, content)
		}
	})

	t.Run("read non-existent file", func(t *testing.T) {
		_, err := osfs.ReadFile("/non/existent/file")
		if err == nil {
			t.Error("ReadFile() error = nil, want error for non-existent file")
		}
	})
}

func TestOSFS_WriteFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "osfs_write_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func(path string) {
		_ = os.RemoveAll(path)
	}(tmpDir)

	osfs := OSFS{}
	content := []byte("written content")

	t.Run("write new file", func(t *testing.T) {
		filePath := filepath.Join(tmpDir, "new_file.txt")
		err := osfs.WriteFile(filePath, content, 0644)
		if err != nil {
			t.Errorf("WriteFile() error = %v, want nil", err)
		}

		data, err := os.ReadFile(filePath)
		if err != nil {
			t.Fatalf("failed to read written file: %v", err)
		}
		if string(data) != string(content) {
			t.Errorf("written content = %q, want %q", data, content)
		}
	})

	t.Run("overwrite existing file", func(t *testing.T) {
		filePath := filepath.Join(tmpDir, "overwrite.txt")
		if err := os.WriteFile(filePath, []byte("old"), 0644); err != nil {
			t.Fatalf("failed to create file: %v", err)
		}

		err := osfs.WriteFile(filePath, content, 0644)
		if err != nil {
			t.Errorf("WriteFile() error = %v, want nil", err)
		}

		data, _ := os.ReadFile(filePath)
		if string(data) != string(content) {
			t.Errorf("overwritten content = %q, want %q", data, content)
		}
	})
}

func TestOSFS_MkdirAll(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "osfs_mkdir_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func(path string) {
		_ = os.RemoveAll(path)
	}(tmpDir)

	osfs := OSFS{}

	t.Run("create nested directories", func(t *testing.T) {
		nestedPath := filepath.Join(tmpDir, "a", "b", "c")
		err := osfs.MkdirAll(nestedPath, 0755)
		if err != nil {
			t.Errorf("MkdirAll() error = %v, want nil", err)
		}

		info, err := os.Stat(nestedPath)
		if err != nil {
			t.Errorf("created directory not found: %v", err)
		}
		if info != nil && !info.IsDir() {
			t.Error("MkdirAll() did not create a directory")
		}
	})

	t.Run("create already existing directory", func(t *testing.T) {
		err := osfs.MkdirAll(tmpDir, 0755)
		if err != nil {
			t.Errorf("MkdirAll() error = %v, want nil for existing dir", err)
		}
	})
}

func TestOSFS_WalkDir(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "osfs_walk_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func(path string) {
		_ = os.RemoveAll(path)
	}(tmpDir)

	subDir := filepath.Join(tmpDir, "subdir")
	_ = os.MkdirAll(subDir, 0755)
	_ = os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("1"), 0644)
	_ = os.WriteFile(filepath.Join(subDir, "file2.txt"), []byte("2"), 0644)

	osfs := OSFS{}

	t.Run("walk directory tree", func(t *testing.T) {
		var visited []string
		err := osfs.WalkDir(tmpDir, func(path string, d iofs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			visited = append(visited, path)
			return nil
		})
		if err != nil {
			t.Errorf("WalkDir() error = %v, want nil", err)
		}
		if len(visited) != 4 {
			t.Errorf("WalkDir() visited %d paths, want 4", len(visited))
		}
	})

	t.Run("walk non-existent directory", func(t *testing.T) {
		err := osfs.WalkDir("/non/existent/path", func(path string, d iofs.DirEntry, err error) error {
			return err
		})
		if err == nil {
			t.Error("WalkDir() error = nil, want error for non-existent path")
		}
	})
}

func TestDefaultFS(t *testing.T) {
	fs := DefaultFS()
	if fs == nil {
		t.Error("DefaultFS() returned nil")
	}

	_, ok := fs.(OSFS)
	if !ok {
		t.Error("DefaultFS() did not return OSFS")
	}
}

func TestOSFS_ImplementsInterfaces(t *testing.T) {
	var _ WritableFS = OSFS{}
	var _ WalkableFS = OSFS{}
}
