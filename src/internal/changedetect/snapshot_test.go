package changedetect

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveLoadLastSnap(t *testing.T) {
	dir := t.TempDir()

	want := map[string]FileSnapshot{
		"main.go": {Path: "main.go", ModTime: 1000, Size: 123, Hash: "abc"},
		"lib/a.go": {Path: "lib/a.go", ModTime: 2000, Size: 456, Hash: "def"},
	}

	if err := SaveLastSnap(dir, want); err != nil {
		t.Fatalf("SaveLastSnap: %v", err)
	}

	got, err := LoadLastSnap(dir)
	if err != nil {
		t.Fatalf("LoadLastSnap: %v", err)
	}

	if len(got) != len(want) {
		t.Fatalf("len mismatch: got %d, want %d", len(got), len(want))
	}
	for k, v := range want {
		g, ok := got[k]
		if !ok {
			t.Fatalf("missing key %q", k)
		}
		if g != v {
			t.Errorf("key %q: got %+v, want %+v", k, g, v)
		}
	}

	// Verify file exists at expected path
	if _, err := os.Stat(filepath.Join(dir, ".chronodraft", "last_snap.json")); err != nil {
		t.Fatalf("expected file not found: %v", err)
	}
}

func TestLoadLastSnapNotExist(t *testing.T) {
	dir := t.TempDir()

	got, err := LoadLastSnap(dir)
	if err == nil {
		t.Fatal("expected error for non-existent file, got nil")
	}
	if got != nil {
		t.Fatalf("expected nil map, got %+v", got)
	}
	if !os.IsNotExist(err) {
		t.Fatalf("expected os.IsNotExist error, got: %v", err)
	}
}
