package gitignore

import (
	"os"
	"path/filepath"
	"testing"
)

func BenchmarkUpsertManagedBlockCreate(b *testing.B) {
	dir := b.TempDir()
	manager := NewManager(dir)
	path := filepath.Join(dir, ".gitignore")
	block := benchmarkManagedBlock([]string{"go", "node"})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			b.Fatalf("remove .gitignore: %v", err)
		}
		action, err := manager.UpsertManagedBlock(block, false)
		if err != nil {
			b.Fatalf("create upsert failed: %v", err)
		}
		if action != FileActionCreated {
			b.Fatalf("expected created action, got %s", action)
		}
	}
}

func BenchmarkUpsertManagedBlockUpdate(b *testing.B) {
	dir := b.TempDir()
	manager := NewManager(dir)
	path := filepath.Join(dir, ".gitignore")
	before := benchmarkManagedBlock([]string{"go"})
	after := benchmarkManagedBlock([]string{"go", "node"})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mustWriteBenchmarkGitignore(b, path, before)
		action, err := manager.UpsertManagedBlock(after, false)
		if err != nil {
			b.Fatalf("update upsert failed: %v", err)
		}
		if action != FileActionUpdated {
			b.Fatalf("expected updated action, got %s", action)
		}
	}
}

func BenchmarkUpsertManagedBlockNoOp(b *testing.B) {
	dir := b.TempDir()
	manager := NewManager(dir)
	path := filepath.Join(dir, ".gitignore")
	block := benchmarkManagedBlock([]string{"go", "node"})
	mustWriteBenchmarkGitignore(b, path, block)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		action, err := manager.UpsertManagedBlock(block, false)
		if err != nil {
			b.Fatalf("no-op upsert failed: %v", err)
		}
		if action != FileActionNoOp {
			b.Fatalf("expected no-op action, got %s", action)
		}
	}
}

func benchmarkManagedBlock(providers []string) string {
	return BuildManagedBlock(providers, "bin/\nnode_modules/\ncoverage.out\n")
}

func mustWriteBenchmarkGitignore(b *testing.B, path, content string) {
	b.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		b.Fatalf("write .gitignore: %v", err)
	}
}
