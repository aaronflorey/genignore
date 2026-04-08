package gitignore

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestUpsertCreatesFileWhenMissing(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	m := NewManager(dir)
	block := BuildManagedBlock([]string{"node"}, "node_modules/")

	action, err := m.UpsertManagedBlock(block, false)
	if err != nil {
		t.Fatalf("upsert returned error: %v", err)
	}
	if action != FileActionCreated {
		t.Fatalf("expected created action, got %s", action)
	}
	content, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	if err != nil {
		t.Fatalf("read .gitignore failed: %v", err)
	}
	if !strings.Contains(string(content), StartMarker) {
		t.Fatalf("expected managed block in file")
	}
}

func TestUpsertPrependsWhenNoMarkers(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, ".gitignore")
	if err := os.WriteFile(path, []byte("*.log\n"), 0o644); err != nil {
		t.Fatalf("seed file failed: %v", err)
	}

	m := NewManager(dir)
	block := BuildManagedBlock([]string{"node"}, "node_modules/")
	if _, err := m.UpsertManagedBlock(block, false); err != nil {
		t.Fatalf("upsert failed: %v", err)
	}
	content, _ := os.ReadFile(path)
	value := string(content)
	if !strings.HasPrefix(value, StartMarker) {
		t.Fatalf("expected block at top")
	}
	if !strings.Contains(value, "*.log") {
		t.Fatalf("expected existing content to remain")
	}
}

func TestUpsertPreservesLeadingBlankLinesWhenNoMarkers(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, ".gitignore")
	seed := "\n\n*.log\n"
	if err := os.WriteFile(path, []byte(seed), 0o644); err != nil {
		t.Fatalf("seed file failed: %v", err)
	}

	m := NewManager(dir)
	block := BuildManagedBlock([]string{"node"}, "node_modules/")
	if _, err := m.UpsertManagedBlock(block, false); err != nil {
		t.Fatalf("upsert failed: %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read .gitignore failed: %v", err)
	}
	want := block + seed
	if string(content) != want {
		t.Fatalf("expected existing content preserved exactly\nwant:\n%q\n got:\n%q", want, string(content))
	}
}

func TestUpsertReplacesOnlyManagedRegion(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, ".gitignore")
	seed := strings.Join([]string{
		"before",
		StartMarker,
		"old",
		EndMarker,
		"after",
		"",
	}, "\n")
	if err := os.WriteFile(path, []byte(seed), 0o644); err != nil {
		t.Fatalf("seed file failed: %v", err)
	}

	m := NewManager(dir)
	block := BuildManagedBlock([]string{"go"}, "bin/")
	if _, err := m.UpsertManagedBlock(block, false); err != nil {
		t.Fatalf("upsert failed: %v", err)
	}
	content, _ := os.ReadFile(path)
	value := string(content)
	if !strings.Contains(value, "before") || !strings.Contains(value, "after") {
		t.Fatalf("expected content outside block untouched")
	}
	if strings.Contains(value, "old") {
		t.Fatalf("expected old block content replaced")
	}
}

func TestUpsertRejectsMalformedMarkers(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, ".gitignore")
	seed := strings.Join([]string{
		"before",
		StartMarker,
		"old",
		"after",
		"",
	}, "\n")
	if err := os.WriteFile(path, []byte(seed), 0o644); err != nil {
		t.Fatalf("seed file failed: %v", err)
	}

	m := NewManager(dir)
	block := BuildManagedBlock([]string{"go"}, "bin/")
	if _, err := m.UpsertManagedBlock(block, false); err == nil {
		t.Fatalf("expected malformed marker error")
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read .gitignore failed: %v", err)
	}
	if string(content) != seed {
		t.Fatalf("malformed input should remain unchanged")
	}
}

func TestUpsertIsIdempotentForEquivalentRun(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, ".gitignore")
	m := NewManager(dir)
	block := BuildManagedBlock([]string{"go", "node"}, "bin/\nnode_modules/\n")

	if _, err := m.UpsertManagedBlock(block, false); err != nil {
		t.Fatalf("first upsert failed: %v", err)
	}
	first, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read first content failed: %v", err)
	}

	if _, err := m.UpsertManagedBlock(block, false); err != nil {
		t.Fatalf("second upsert failed: %v", err)
	}
	second, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read second content failed: %v", err)
	}

	if string(first) != string(second) {
		t.Fatalf("expected equivalent rerun to be byte-identical")
	}
	firstInfo, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat first content failed: %v", err)
	}
	time.Sleep(20 * time.Millisecond)
	if _, err := m.UpsertManagedBlock(block, false); err != nil {
		t.Fatalf("third upsert failed: %v", err)
	}
	secondInfo, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat second content failed: %v", err)
	}
	if !firstInfo.ModTime().Equal(secondInfo.ModTime()) {
		t.Fatalf("expected equivalent rerun to avoid rewriting file")
	}
}

func TestUpsertDryRunLeavesExistingFileByteExact(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, ".gitignore")
	seed := strings.Join([]string{
		"# user content",
		StartMarker,
		"old",
		EndMarker,
		"*.log",
		"",
	}, "\n")
	if err := os.WriteFile(path, []byte(seed), 0o644); err != nil {
		t.Fatalf("seed file failed: %v", err)
	}

	m := NewManager(dir)
	block := BuildManagedBlock([]string{"go"}, "bin/")
	action, err := m.UpsertManagedBlock(block, true)
	if err != nil {
		t.Fatalf("dry-run upsert failed: %v", err)
	}
	if action != FileActionDryRun {
		t.Fatalf("unexpected action: %s", action)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read .gitignore failed: %v", err)
	}
	if string(content) != seed {
		t.Fatalf("expected dry-run to preserve file exactly")
	}
}
