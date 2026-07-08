package signalfile

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
	"time"
)

func TestReadFSRejectsUnsafePaths(t *testing.T) {
	t.Parallel()

	tests := []string{
		"",
		"/etc/passwd",
		"//server/share/file",
		"../package.json",
		`..\\package.json`,
		"../frontend\\package.json",
		`..\\frontend/package.json`,
		`C:\\package.json`,
		"C:/package.json",
		`\\server\\share\\file`,
	}

	for _, name := range tests {
		_, err := ReadFS(fstest.MapFS{}, name)
		if !errors.Is(err, ErrUnsafePath) {
			t.Fatalf("ReadFS(%q) error = %v, want ErrUnsafePath", name, err)
		}
		if !IsSafeSkip(err) {
			t.Fatalf("IsSafeSkip(%q) = false, want true", name)
		}
	}
}

func TestReadFSRejectsDirectories(t *testing.T) {
	t.Parallel()

	_, err := ReadFS(fstest.MapFS{
		"signals": &fstest.MapFile{Mode: fs.ModeDir},
	}, "signals")
	if !errors.Is(err, ErrDirectory) {
		t.Fatalf("ReadFS() error = %v, want ErrDirectory", err)
	}
	if IsSafeSkip(err) {
		t.Fatal("IsSafeSkip(directory error) = true, want false")
	}
}

func TestReadFSReadsAtMostConfiguredLimitPlusSentinel(t *testing.T) {
	t.Parallel()

	file := &countingFile{data: make([]byte, MaxReadBytes+128)}
	_, err := ReadFS(countingFS{file: file}, "package.json")
	if !errors.Is(err, ErrTooLarge) {
		t.Fatalf("ReadFS() error = %v, want ErrTooLarge", err)
	}
	if got, want := int64(file.bytesRead), MaxReadBytes+1; got != want {
		t.Fatalf("ReadFS() consumed %d bytes, want %d", got, want)
	}
	if !IsSafeSkip(err) {
		t.Fatal("IsSafeSkip(ErrTooLarge) = false, want true")
	}
}

func TestReadOSRejectsSymlinkedSignalFile(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	target := filepath.Join(root, "outside.txt")
	if err := os.WriteFile(target, []byte("vue"), 0o644); err != nil {
		t.Fatalf("write target: %v", err)
	}
	link := filepath.Join(root, "package.json")
	if err := os.Symlink(target, link); err != nil {
		if errors.Is(err, fs.ErrPermission) || errors.Is(err, os.ErrPermission) {
			t.Skipf("symlink unavailable: %v", err)
		}
		t.Fatalf("symlink package.json: %v", err)
	}

	_, err := ReadOS(root, "package.json")
	if !errors.Is(err, ErrSymlink) {
		t.Fatalf("ReadOS() error = %v, want ErrSymlink", err)
	}
	if !IsSafeSkip(err) {
		t.Fatal("IsSafeSkip(ErrSymlink) = false, want true")
	}
}

func TestReadOSRejectsSymlinkedPathComponent(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	targetDir := filepath.Join(root, "outside")
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		t.Fatalf("mkdir target dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(targetDir, "package.json"), []byte("vue"), 0o644); err != nil {
		t.Fatalf("write target package.json: %v", err)
	}
	if err := os.Symlink(targetDir, filepath.Join(root, "frontend")); err != nil {
		if errors.Is(err, fs.ErrPermission) || errors.Is(err, os.ErrPermission) {
			t.Skipf("symlink unavailable: %v", err)
		}
		t.Fatalf("symlink frontend: %v", err)
	}

	_, err := ReadOS(root, "frontend/package.json")
	if !errors.Is(err, ErrSymlink) {
		t.Fatalf("ReadOS() error = %v, want ErrSymlink", err)
	}
}

func TestReadFSReturnsUnderlyingReadErrors(t *testing.T) {
	t.Parallel()

	boom := errors.New("boom")
	_, err := ReadFS(errorFS{err: boom}, "package.json")
	if !errors.Is(err, boom) {
		t.Fatalf("ReadFS() error = %v, want %v", err, boom)
	}
	if IsSafeSkip(err) {
		t.Fatal("IsSafeSkip(unexpected error) = true, want false")
	}
}

type countingFS struct {
	file *countingFile
}

func (f countingFS) Open(name string) (fs.File, error) {
	if name != "package.json" {
		return nil, fs.ErrNotExist
	}
	f.file.offset = 0
	f.file.bytesRead = 0
	return f.file, nil
}

type countingFile struct {
	data      []byte
	offset    int
	bytesRead int
}

func (f *countingFile) Stat() (fs.FileInfo, error) {
	return staticFileInfo{size: int64(len(f.data))}, nil
}
func (f *countingFile) Close() error { return nil }

func (f *countingFile) Read(p []byte) (int, error) {
	if f.offset >= len(f.data) {
		return 0, io.EOF
	}
	n := copy(p, f.data[f.offset:])
	f.offset += n
	f.bytesRead += n
	return n, nil
}

type errorFS struct {
	err error
}

func (f errorFS) Open(name string) (fs.File, error) {
	return &errorFile{err: f.err}, nil
}

type errorFile struct {
	err error
}

func (f *errorFile) Stat() (fs.FileInfo, error) { return staticFileInfo{}, nil }
func (f *errorFile) Close() error               { return nil }
func (f *errorFile) Read([]byte) (int, error)   { return 0, f.err }

type staticFileInfo struct {
	size int64
}

func (info staticFileInfo) Name() string       { return "package.json" }
func (info staticFileInfo) Size() int64        { return info.size }
func (info staticFileInfo) Mode() fs.FileMode  { return 0 }
func (info staticFileInfo) ModTime() time.Time { return time.Time{} }
func (info staticFileInfo) IsDir() bool        { return false }
func (info staticFileInfo) Sys() any           { return nil }
