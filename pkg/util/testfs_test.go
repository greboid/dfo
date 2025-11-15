package util

import (
	"errors"
	"io"
	"io/fs"
	"testing"
	"testing/fstest"
)

func TestNewTestFS(t *testing.T) {
	tfs := NewTestFS()
	if tfs == nil {
		t.Fatal("NewTestFS() returned nil")
	}
	if tfs.MapFS == nil {
		t.Error("NewTestFS() MapFS is nil")
	}
}

func TestTestFS_Open(t *testing.T) {
	tfs := NewTestFS()
	tfs.MapFS["test.txt"] = &fstest.MapFile{Data: []byte("content")}

	t.Run("open existing file", func(t *testing.T) {
		f, err := tfs.Open("test.txt")
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
		_, err := tfs.Open("nonexistent.txt")
		if err == nil {
			t.Error("Open() error = nil, want error")
		}
	})
}

func TestTestFS_Stat(t *testing.T) {
	tfs := NewTestFS()
	tfs.MapFS["file.txt"] = &fstest.MapFile{Data: []byte("data"), Mode: 0644}
	tfs.MapFS["dir"] = &fstest.MapFile{Mode: fs.ModeDir | 0755}

	t.Run("stat file", func(t *testing.T) {
		info, err := tfs.Stat("file.txt")
		if err != nil {
			t.Errorf("Stat() error = %v, want nil", err)
		}
		if info.IsDir() {
			t.Error("Stat() file reported as directory")
		}
		if info.Size() != 4 {
			t.Errorf("Stat() size = %d, want 4", info.Size())
		}
	})

	t.Run("stat directory", func(t *testing.T) {
		info, err := tfs.Stat("dir")
		if err != nil {
			t.Errorf("Stat() error = %v, want nil", err)
		}
		if !info.IsDir() {
			t.Error("Stat() directory not reported as directory")
		}
	})

	t.Run("stat non-existent", func(t *testing.T) {
		_, err := tfs.Stat("nonexistent")
		if err == nil {
			t.Error("Stat() error = nil, want error")
		}
	})
}

func TestTestFS_ReadFile(t *testing.T) {
	tfs := NewTestFS()
	content := []byte("file content")
	tfs.MapFS["readable.txt"] = &fstest.MapFile{Data: content}

	t.Run("read existing file", func(t *testing.T) {
		data, err := tfs.ReadFile("readable.txt")
		if err != nil {
			t.Errorf("ReadFile() error = %v, want nil", err)
		}
		if string(data) != string(content) {
			t.Errorf("ReadFile() = %q, want %q", data, content)
		}
	})

	t.Run("read non-existent file", func(t *testing.T) {
		_, err := tfs.ReadFile("nonexistent.txt")
		if err == nil {
			t.Error("ReadFile() error = nil, want error")
		}
	})
}

func TestTestFS_WriteFile(t *testing.T) {
	t.Run("write to root", func(t *testing.T) {
		tfs := NewTestFS()
		content := []byte("new content")

		err := tfs.WriteFile("newfile.txt", content, 0644)
		if err != nil {
			t.Errorf("WriteFile() error = %v, want nil", err)
		}

		data, err := tfs.ReadFile("newfile.txt")
		if err != nil {
			t.Fatalf("ReadFile() after write error = %v", err)
		}
		if string(data) != string(content) {
			t.Errorf("written content = %q, want %q", data, content)
		}
	})

	t.Run("write creates parent directories", func(t *testing.T) {
		tfs := NewTestFS()
		content := []byte("nested content")

		err := tfs.WriteFile("a/b/c/file.txt", content, 0644)
		if err != nil {
			t.Errorf("WriteFile() error = %v, want nil", err)
		}

		info, err := tfs.Stat("a")
		if err != nil {
			t.Errorf("parent dir 'a' not created: %v", err)
		}
		if info != nil && !info.IsDir() {
			t.Error("'a' should be a directory")
		}

		info, err = tfs.Stat("a/b")
		if err != nil {
			t.Errorf("parent dir 'a/b' not created: %v", err)
		}

		info, err = tfs.Stat("a/b/c")
		if err != nil {
			t.Errorf("parent dir 'a/b/c' not created: %v", err)
		}
	})

	t.Run("overwrite existing file", func(t *testing.T) {
		tfs := NewTestFS()
		tfs.MapFS["existing.txt"] = &fstest.MapFile{Data: []byte("old")}

		newContent := []byte("new")
		err := tfs.WriteFile("existing.txt", newContent, 0644)
		if err != nil {
			t.Errorf("WriteFile() error = %v, want nil", err)
		}

		data, _ := tfs.ReadFile("existing.txt")
		if string(data) != "new" {
			t.Errorf("overwritten content = %q, want %q", data, "new")
		}
	})
}

func TestTestFS_MkdirAll(t *testing.T) {
	t.Run("create nested directories", func(t *testing.T) {
		tfs := NewTestFS()

		err := tfs.MkdirAll("x/y/z", 0755)
		if err != nil {
			t.Errorf("MkdirAll() error = %v, want nil", err)
		}

		for _, dir := range []string{"x", "x/y", "x/y/z"} {
			info, err := tfs.Stat(dir)
			if err != nil {
				t.Errorf("MkdirAll() did not create %q: %v", dir, err)
			}
			if info != nil && !info.IsDir() {
				t.Errorf("%q should be a directory", dir)
			}
		}
	})

	t.Run("create single directory", func(t *testing.T) {
		tfs := NewTestFS()

		err := tfs.MkdirAll("single", 0755)
		if err != nil {
			t.Errorf("MkdirAll() error = %v, want nil", err)
		}

		info, err := tfs.Stat("single")
		if err != nil {
			t.Errorf("directory not created: %v", err)
		}
		if info != nil && !info.IsDir() {
			t.Error("should be a directory")
		}
	})
}

func TestTestFS_ensureDir(t *testing.T) {
	t.Run("handles empty parts", func(t *testing.T) {
		tfs := NewTestFS()
		tfs.ensureDir("./a/b")

		if _, exists := tfs.MapFS["a"]; !exists {
			t.Error("'a' directory not created")
		}
		if _, exists := tfs.MapFS["a/b"]; !exists {
			t.Error("'a/b' directory not created")
		}
	})

	t.Run("does not overwrite existing", func(t *testing.T) {
		tfs := NewTestFS()
		tfs.MapFS["existing"] = &fstest.MapFile{Mode: fs.ModeDir | 0700}

		tfs.ensureDir("existing/new")

		if tfs.MapFS["existing"].Mode != fs.ModeDir|0700 {
			t.Error("ensureDir() overwrote existing directory")
		}
	})
}

func TestTestFS_WalkDir(t *testing.T) {
	t.Run("walk from root", func(t *testing.T) {
		tfs := NewTestFS()
		tfs.MapFS["a"] = &fstest.MapFile{Mode: fs.ModeDir | 0755}
		tfs.MapFS["a/file.txt"] = &fstest.MapFile{Data: []byte("1")}
		tfs.MapFS["b.txt"] = &fstest.MapFile{Data: []byte("2")}

		var visited []string
		err := tfs.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
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
			t.Errorf("WalkDir() visited %d paths, want 4: %v", len(visited), visited)
		}
	})

	t.Run("walk from empty root", func(t *testing.T) {
		tfs := NewTestFS()
		tfs.MapFS["file.txt"] = &fstest.MapFile{Data: []byte("data")}

		var visited []string
		err := tfs.WalkDir("", func(path string, d fs.DirEntry, err error) error {
			visited = append(visited, path)
			return nil
		})

		if err != nil {
			t.Errorf("WalkDir() error = %v, want nil", err)
		}
		if len(visited) != 2 {
			t.Errorf("WalkDir() visited %d paths, want 2: %v", len(visited), visited)
		}
	})

	t.Run("walk from subdirectory", func(t *testing.T) {
		tfs := NewTestFS()
		tfs.MapFS["dir"] = &fstest.MapFile{Mode: fs.ModeDir | 0755}
		tfs.MapFS["dir/a.txt"] = &fstest.MapFile{Data: []byte("a")}
		tfs.MapFS["dir/b.txt"] = &fstest.MapFile{Data: []byte("b")}
		tfs.MapFS["other.txt"] = &fstest.MapFile{Data: []byte("other")}

		var visited []string
		err := tfs.WalkDir("dir", func(path string, d fs.DirEntry, err error) error {
			visited = append(visited, path)
			return nil
		})

		if err != nil {
			t.Errorf("WalkDir() error = %v, want nil", err)
		}
		if len(visited) != 3 {
			t.Errorf("WalkDir() visited %d paths, want 3: %v", len(visited), visited)
		}
	})

	t.Run("SkipDir on root", func(t *testing.T) {
		tfs := NewTestFS()
		tfs.MapFS["file.txt"] = &fstest.MapFile{Data: []byte("data")}

		var visited []string
		err := tfs.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
			visited = append(visited, path)
			if path == "." {
				return fs.SkipDir
			}
			return nil
		})

		if err != nil {
			t.Errorf("WalkDir() error = %v, want nil", err)
		}
		if len(visited) != 1 {
			t.Errorf("WalkDir() with SkipDir visited %d paths, want 1", len(visited))
		}
	})

	t.Run("SkipDir on subdirectory root", func(t *testing.T) {
		tfs := NewTestFS()
		tfs.MapFS["subdir"] = &fstest.MapFile{Mode: fs.ModeDir | 0755}
		tfs.MapFS["subdir/file.txt"] = &fstest.MapFile{Data: []byte("data")}

		var visited []string
		err := tfs.WalkDir("subdir", func(path string, d fs.DirEntry, err error) error {
			visited = append(visited, path)
			if path == "subdir" {
				return fs.SkipDir
			}
			return nil
		})

		if err != nil {
			t.Errorf("WalkDir() error = %v, want nil", err)
		}
		if len(visited) != 1 {
			t.Errorf("WalkDir() with SkipDir visited %d paths, want 1: %v", len(visited), visited)
		}
	})

	t.Run("SkipDir on nested directory", func(t *testing.T) {
		tfs := NewTestFS()
		tfs.MapFS["a"] = &fstest.MapFile{Mode: fs.ModeDir | 0755}
		tfs.MapFS["a/skip"] = &fstest.MapFile{Mode: fs.ModeDir | 0755}
		tfs.MapFS["a/skip/file.txt"] = &fstest.MapFile{Data: []byte("hidden")}
		tfs.MapFS["a/keep.txt"] = &fstest.MapFile{Data: []byte("visible")}

		var visited []string
		err := tfs.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
			visited = append(visited, path)
			if path == "a/skip" {
				return fs.SkipDir
			}
			return nil
		})

		if err != nil {
			t.Errorf("WalkDir() error = %v, want nil", err)
		}
		for _, v := range visited {
			if v == "a/skip/file.txt" {
				t.Error("WalkDir() visited path that should be skipped")
			}
		}
	})

	t.Run("error from callback", func(t *testing.T) {
		tfs := NewTestFS()
		tfs.MapFS["file.txt"] = &fstest.MapFile{Data: []byte("data")}

		testErr := errors.New("test error")
		err := tfs.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
			if path == "file.txt" {
				return testErr
			}
			return nil
		})

		if !errors.Is(err, testErr) {
			t.Errorf("WalkDir() error = %v, want %v", err, testErr)
		}
	})

	t.Run("walk non-existent root", func(t *testing.T) {
		tfs := NewTestFS()

		var callbackErr error
		err := tfs.WalkDir("nonexistent", func(path string, d fs.DirEntry, err error) error {
			callbackErr = err
			return err
		})

		if err == nil {
			t.Error("WalkDir() error = nil, want error for non-existent path")
		}
		if callbackErr == nil {
			t.Error("callback should receive error for non-existent path")
		}
	})
}

func TestTestFS_visitPath(t *testing.T) {
	t.Run("visit existing file", func(t *testing.T) {
		tfs := NewTestFS()
		tfs.MapFS["file.txt"] = &fstest.MapFile{Data: []byte("data")}

		var visited string
		skip, skippedDir, err := tfs.visitPath("file.txt", func(path string, d fs.DirEntry, err error) error {
			visited = path
			return nil
		})

		if err != nil {
			t.Errorf("visitPath() error = %v, want nil", err)
		}
		if skip {
			t.Error("visitPath() skip = true, want false")
		}
		if skippedDir != "" {
			t.Errorf("visitPath() skippedDir = %q, want empty", skippedDir)
		}
		if visited != "file.txt" {
			t.Errorf("visitPath() visited = %q, want %q", visited, "file.txt")
		}
	})

	t.Run("visit with SkipDir on directory", func(t *testing.T) {
		tfs := NewTestFS()
		tfs.MapFS["dir"] = &fstest.MapFile{Mode: fs.ModeDir | 0755}

		skip, skippedDir, err := tfs.visitPath("dir", func(path string, d fs.DirEntry, err error) error {
			return fs.SkipDir
		})

		if err != nil {
			t.Errorf("visitPath() error = %v, want nil", err)
		}
		if !skip {
			t.Error("visitPath() skip = false, want true for SkipDir")
		}
		if skippedDir != "dir" {
			t.Errorf("visitPath() skippedDir = %q, want %q", skippedDir, "dir")
		}
	})

	t.Run("visit with SkipDir on file", func(t *testing.T) {
		tfs := NewTestFS()
		tfs.MapFS["file.txt"] = &fstest.MapFile{Data: []byte("data")}

		skip, skippedDir, err := tfs.visitPath("file.txt", func(path string, d fs.DirEntry, err error) error {
			return fs.SkipDir
		})

		if err != nil {
			t.Errorf("visitPath() error = %v, want nil", err)
		}
		if !skip {
			t.Error("visitPath() skip = false, want true for SkipDir")
		}
		if skippedDir != "" {
			t.Errorf("visitPath() skippedDir = %q, want empty for file", skippedDir)
		}
	})

	t.Run("visit with error", func(t *testing.T) {
		tfs := NewTestFS()
		tfs.MapFS["file.txt"] = &fstest.MapFile{Data: []byte("data")}

		testErr := errors.New("callback error")
		skip, skippedDir, err := tfs.visitPath("file.txt", func(path string, d fs.DirEntry, err error) error {
			return testErr
		})

		if !errors.Is(err, testErr) {
			t.Errorf("visitPath() error = %v, want %v", err, testErr)
		}
		if skip {
			t.Error("visitPath() skip = true, want false on error")
		}
		if skippedDir != "" {
			t.Errorf("visitPath() skippedDir = %q, want empty on error", skippedDir)
		}
	})

	t.Run("visit non-existent path with SkipDir", func(t *testing.T) {
		tfs := NewTestFS()

		skip, skippedDir, err := tfs.visitPath("nonexistent", func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return fs.SkipDir
			}
			return nil
		})

		if err != nil {
			t.Errorf("visitPath() error = %v, want nil when callback returns SkipDir", err)
		}
		if !skip {
			t.Error("visitPath() skip = false, want true")
		}
		if skippedDir != "" {
			t.Errorf("visitPath() skippedDir = %q, want empty for non-existent", skippedDir)
		}
	})

	t.Run("visit non-existent path with error", func(t *testing.T) {
		tfs := NewTestFS()

		testErr := errors.New("propagated error")
		skip, skippedDir, err := tfs.visitPath("nonexistent", func(path string, d fs.DirEntry, err error) error {
			return testErr
		})

		if !errors.Is(err, testErr) {
			t.Errorf("visitPath() error = %v, want %v", err, testErr)
		}
		if skip {
			t.Error("visitPath() skip = true, want false on error")
		}
		if skippedDir != "" {
			t.Errorf("visitPath() skippedDir = %q, want empty on error", skippedDir)
		}
	})
}

func TestMapDirEntry(t *testing.T) {
	t.Run("file entry", func(t *testing.T) {
		entry := &mapDirEntry{name: "file.txt", isDir: false}

		if entry.Name() != "file.txt" {
			t.Errorf("Name() = %q, want %q", entry.Name(), "file.txt")
		}
		if entry.IsDir() {
			t.Error("IsDir() = true, want false")
		}
		if entry.Type() != 0 {
			t.Errorf("Type() = %v, want 0", entry.Type())
		}
	})

	t.Run("directory entry", func(t *testing.T) {
		entry := &mapDirEntry{name: "dir", isDir: true}

		if entry.Name() != "dir" {
			t.Errorf("Name() = %q, want %q", entry.Name(), "dir")
		}
		if !entry.IsDir() {
			t.Error("IsDir() = false, want true")
		}
		if entry.Type() != fs.ModeDir {
			t.Errorf("Type() = %v, want %v", entry.Type(), fs.ModeDir)
		}
	})

	t.Run("Info returns stored info", func(t *testing.T) {
		tfs := NewTestFS()
		tfs.MapFS["test.txt"] = &fstest.MapFile{Data: []byte("test")}
		info, _ := tfs.Stat("test.txt")

		entry := &mapDirEntry{name: "test.txt", isDir: false, info: info}

		gotInfo, err := entry.Info()
		if err != nil {
			t.Errorf("Info() error = %v, want nil", err)
		}
		if gotInfo != info {
			t.Error("Info() returned different FileInfo")
		}
	})

	t.Run("Info returns nil when not set", func(t *testing.T) {
		entry := &mapDirEntry{name: "test.txt", isDir: false}

		info, err := entry.Info()
		if err != nil {
			t.Errorf("Info() error = %v, want nil", err)
		}
		if info != nil {
			t.Error("Info() should return nil when not set")
		}
	})
}

func TestTestFS_ImplementsInterfaces(t *testing.T) {
	var _ WritableFS = &TestFS{}
	var _ WalkableFS = &TestFS{}
}

func TestTestFS_OpenAndRead(t *testing.T) {
	tfs := NewTestFS()
	content := []byte("hello world")
	tfs.MapFS["test.txt"] = &fstest.MapFile{Data: content}

	f, err := tfs.Open("test.txt")
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer func() { _ = f.Close() }()

	data, err := io.ReadAll(f)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if string(data) != string(content) {
		t.Errorf("Read content = %q, want %q", data, content)
	}
}
