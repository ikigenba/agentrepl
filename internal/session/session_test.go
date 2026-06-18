package session

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestIDDeterministicAndOpenTargetsTimestampPath(t *testing.T) {
	// R-8GF4-LRYU
	now := time.Date(2026, 6, 18, 14, 5, 6, 789123000, time.UTC)
	wantID := "20260618T140506.789123"

	firstID := ID(now)
	secondID := ID(now)
	if firstID != wantID {
		t.Fatalf("ID() = %q, want %q", firstID, wantID)
	}
	if secondID != firstID {
		t.Fatalf("ID() is not deterministic for fixed time: got %q then %q", firstID, secondID)
	}

	dir := t.TempDir()
	file, gotID, err := Open(dir, now)
	if err != nil {
		t.Fatalf("Open() returned error: %v", err)
	}
	defer file.Close()

	if gotID != wantID {
		t.Fatalf("Open() id = %q, want %q", gotID, wantID)
	}
	wantPath := filepath.Join(dir, wantID+".jsonl")
	if file.Name() != wantPath {
		t.Fatalf("Open() file path = %q, want %q", file.Name(), wantPath)
	}
	if _, err := os.Stat(wantPath); err != nil {
		t.Fatalf("Open() did not create target file %q: %v", wantPath, err)
	}
}

func TestOpenCreatesMissingDirAndOpensUnbufferedWritableTruncatedFile(t *testing.T) {
	// R-8HN0-ZJPJ
	now := time.Date(2026, 6, 18, 14, 5, 6, 1_000, time.UTC)
	dir := filepath.Join(t.TempDir(), "missing", "nested")
	path := filepath.Join(dir, ID(now)+".jsonl")

	file, _, err := Open(dir, now)
	if err != nil {
		t.Fatalf("Open() returned error for missing dir: %v", err)
	}
	if _, err := file.WriteString("old content"); err != nil {
		t.Fatalf("initial write failed: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("initial close failed: %v", err)
	}

	file, _, err = Open(dir, now)
	if err != nil {
		t.Fatalf("Open() returned error for existing file: %v", err)
	}
	defer file.Close()

	if info, err := os.Stat(dir); err != nil {
		t.Fatalf("Open() did not create dir %q: %v", dir, err)
	} else if !info.IsDir() {
		t.Fatalf("Open() target dir %q is not a directory", dir)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading truncated file failed: %v", err)
	}
	if string(got) != "" {
		t.Fatalf("Open() did not truncate existing file, content = %q", got)
	}

	if _, err := file.WriteString("new content\n"); err != nil {
		t.Fatalf("write through returned file failed: %v", err)
	}
	got, err = os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading open file failed: %v", err)
	}
	if string(got) != "new content\n" {
		t.Fatalf("write was not visible through path before Close(), content = %q", got)
	}
}
