package signalfile

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"strings"
)

// MaxReadBytes bounds repository signal file reads to a small, deterministic
// size. Reads consume at most MaxReadBytes plus one sentinel byte to detect
// oversized files without loading the full body.
const MaxReadBytes int64 = 1 << 20

var (
	ErrUnsafePath = errors.New("unsafe signal file path")
	ErrSymlink    = errors.New("signal file path contains symlink")
	ErrDirectory  = errors.New("signal file path is a directory")
	ErrTooLarge   = errors.New("signal file exceeds safe read limit")
)

type lstatFS interface {
	Lstat(name string) (fs.FileInfo, error)
}

type RootFS struct {
	root *os.Root
	fs   fs.FS
}

func OpenRootFS(dir string) (*RootFS, error) {
	root, err := os.OpenRoot(dir)
	if err != nil {
		return nil, err
	}

	return &RootFS{root: root, fs: root.FS()}, nil
}

func (r *RootFS) Close() error {
	return r.root.Close()
}

func (r *RootFS) Open(name string) (fs.File, error) {
	return r.fs.Open(name)
}

func (r *RootFS) ReadDir(name string) ([]fs.DirEntry, error) {
	return fs.ReadDir(r.fs, name)
}

func (r *RootFS) Stat(name string) (fs.FileInfo, error) {
	return fs.Stat(r.fs, name)
}

func (r *RootFS) Lstat(name string) (fs.FileInfo, error) {
	return r.root.Lstat(name)
}

func ReadOS(rootDir, name string) ([]byte, error) {
	root, err := OpenRootFS(rootDir)
	if err != nil {
		return nil, err
	}
	defer root.Close()

	return ReadFS(root, name)
}

func ReadFS(root fs.FS, name string) ([]byte, error) {
	cleaned, err := cleanRelativePath(name)
	if err != nil {
		return nil, err
	}

	if rootWithLstat, ok := root.(lstatFS); ok {
		if err := rejectSymlinks(rootWithLstat, cleaned); err != nil {
			return nil, err
		}
	}

	file, err := root.Open(cleaned)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		return nil, fmt.Errorf("%w: %s", ErrDirectory, cleaned)
	}

	content, err := io.ReadAll(io.LimitReader(file, MaxReadBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(content)) > MaxReadBytes {
		return nil, fmt.Errorf("%w: %s", ErrTooLarge, cleaned)
	}

	return content, nil
}

func IsSafeSkip(err error) bool {
	return errors.Is(err, ErrUnsafePath) ||
		errors.Is(err, ErrSymlink) ||
		errors.Is(err, ErrTooLarge)
}

func cleanRelativePath(name string) (string, error) {
	normalized := strings.ReplaceAll(strings.TrimSpace(name), "\\", "/")
	if normalized == "" {
		return "", fmt.Errorf("%w: empty path", ErrUnsafePath)
	}
	if strings.HasPrefix(normalized, "/") || strings.HasPrefix(normalized, "//") {
		return "", fmt.Errorf("%w: %s", ErrUnsafePath, name)
	}
	if len(normalized) >= 2 && normalized[1] == ':' {
		return "", fmt.Errorf("%w: %s", ErrUnsafePath, name)
	}

	cleaned := path.Clean(normalized)
	if !fs.ValidPath(cleaned) {
		return "", fmt.Errorf("%w: %s", ErrUnsafePath, name)
	}

	return cleaned, nil
}

func rejectSymlinks(root lstatFS, name string) error {
	parts := strings.Split(name, "/")
	current := parts[0]
	for i, part := range parts {
		if i == 0 {
			current = part
		} else {
			current += "/" + part
		}

		info, err := root.Lstat(current)
		if err != nil {
			return err
		}
		if info.Mode()&fs.ModeSymlink != 0 {
			return fmt.Errorf("%w: %s", ErrSymlink, current)
		}
	}

	return nil
}
