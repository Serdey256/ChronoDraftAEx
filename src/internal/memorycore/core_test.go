package memorycore

import (
	"ChronoDraftAEx/pkg/models"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewMemoryCoreDoesNotCreateAgentsFile(t *testing.T) {
	projectRoot := t.TempDir()

	core, err := NewMemoryCore(projectRoot, "", "", "")
	if err != nil {
		t.Fatalf("NewMemoryCore failed: %v", err)
	}
	t.Cleanup(func() { _ = core.Close() })

	agentsPath := filepath.Join(projectRoot, "AGENTS.md")
	if _, err := os.Stat(agentsPath); !os.IsNotExist(err) {
		t.Fatalf("expected no AGENTS.md to be created on startup, got err=%v", err)
	}
}

func TestScaffoldProjectDoesNotCreateAgentsFile(t *testing.T) {
	projectRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(projectRoot, "main.go"), []byte("package main\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	core, err := NewMemoryCore(projectRoot, "", "", "")
	if err != nil {
		t.Fatalf("NewMemoryCore failed: %v", err)
	}
	t.Cleanup(func() { _ = core.Close() })

	agentsPath := filepath.Join(projectRoot, "AGENTS.md")
	_ = os.Remove(agentsPath)

	if _, err := core.ScaffoldProject(); err != nil {
		t.Fatalf("ScaffoldProject failed: %v", err)
	}

	if _, err := os.Stat(agentsPath); !os.IsNotExist(err) {
		t.Fatalf("expected no AGENTS.md to be created by scaffold, got err=%v", err)
	}
}

func TestGenerateAgentsMDDoesNotCreateAgentsFile(t *testing.T) {
	projectRoot := t.TempDir()

	core, err := NewMemoryCore(projectRoot, "", "", "")
	if err != nil {
		t.Fatalf("NewMemoryCore failed: %v", err)
	}
	t.Cleanup(func() { _ = core.Close() })

	agentsPath := filepath.Join(projectRoot, "AGENTS.md")
	_ = os.Remove(agentsPath)

	if err := core.GenerateAgentsMD(); err != nil {
		t.Fatalf("GenerateAgentsMD failed: %v", err)
	}

	if _, err := os.Stat(agentsPath); !os.IsNotExist(err) {
		t.Fatalf("expected no AGENTS.md to be created by GenerateAgentsMD, got err=%v", err)
	}
}

func TestImportGitHistoryStoresChangedFiles(t *testing.T) {
	projectRoot := t.TempDir()
	writeFile(t, filepath.Join(projectRoot, "main.go"), "package main\nfunc main() {}\n")
	gitRun(t, projectRoot, "init")
	gitRun(t, projectRoot, "config", "user.email", "test@example.com")
	gitRun(t, projectRoot, "config", "user.name", "ChronoDraft Test")
	gitRun(t, projectRoot, "add", ".")
	gitRun(t, projectRoot, "commit", "-m", "init")

	core, err := NewMemoryCore(projectRoot, "", "", "")
	if err != nil {
		t.Fatalf("NewMemoryCore failed: %v", err)
	}
	t.Cleanup(func() { _ = core.Close() })

	if err := core.ImportGitHistory(10); err != nil {
		t.Fatalf("ImportGitHistory failed: %v", err)
	}

	commits, err := core.ListCommits(10)
	if err != nil {
		t.Fatalf("ListCommits failed: %v", err)
	}
	if len(commits) == 0 {
		t.Fatal("expected imported commits")
	}
	if !strings.Contains(commits[0].Files, "main.go") {
		t.Fatalf("expected imported commit files to include main.go, got %q", commits[0].Files)
	}
	if commits[0].Insertions == 0 {
		t.Fatalf("expected imported commit insertions to be populated, got %d", commits[0].Insertions)
	}
}

func TestScaffoldProjectCreatesImportEdges(t *testing.T) {
	projectRoot := t.TempDir()
	writeFile(t, filepath.Join(projectRoot, "src", "util.ts"), "export const answer = 42\n")
	writeFile(t, filepath.Join(projectRoot, "src", "main.ts"), "import { answer } from './util'\nconsole.log(answer)\n")

	core, err := NewMemoryCore(projectRoot, "", "", "")
	if err != nil {
		t.Fatalf("NewMemoryCore failed: %v", err)
	}
	t.Cleanup(func() { _ = core.Close() })

	if _, err := core.ScaffoldProject(); err != nil {
		t.Fatalf("ScaffoldProject failed: %v", err)
	}

	_, edges, err := core.GetGraphData(100)
	if err != nil {
		t.Fatalf("GetGraphData failed: %v", err)
	}

	for _, edge := range edges {
		if edge.Relation == "IMPORTS" && edge.SourceID == "file:src/main.ts" && edge.TargetID == "file:src/util.ts" {
			return
		}
	}
	t.Fatal("expected IMPORTS edge file:src/main.ts -> file:src/util.ts")
}

func TestSearchGraphReturnsStructureForDirectoryQuery(t *testing.T) {
	projectRoot := t.TempDir()
	writeFile(t, filepath.Join(projectRoot, "src", "util.ts"), "export const answer = 42\n")

	core, err := NewMemoryCore(projectRoot, "", "", "")
	if err != nil {
		t.Fatalf("NewMemoryCore failed: %v", err)
	}
	t.Cleanup(func() { _ = core.Close() })

	if _, err := core.ScaffoldProject(); err != nil {
		t.Fatalf("ScaffoldProject failed: %v", err)
	}

	nodes, _, err := core.SearchGraph("src", 5)
	if err != nil {
		t.Fatalf("SearchGraph failed: %v", err)
	}
	if len(nodes) == 0 {
		t.Fatal("expected structure graph nodes for directory query")
	}
	for _, n := range nodes {
		if n.ID == "dir:src" {
			return
		}
	}
	t.Fatalf("expected returned nodes to include dir:src, got %#v", nodes)
}

func TestDeleteEntryRemovesAndRestoreReindexesEntry(t *testing.T) {
	projectRoot := t.TempDir()
	core, err := NewMemoryCore(projectRoot, "", "", "")
	if err != nil {
		t.Fatalf("NewMemoryCore failed: %v", err)
	}
	t.Cleanup(func() { _ = core.Close() })

	entry := &models.StructuredEntry{
		ID:             "entry-delete-test",
		Timestamp:      time.Now(),
		SessionID:      "session-delete-test",
		Summary:        "dashboard delete button change",
		DesignDecision: "store deletion is reversible from the dashboard",
		ImpactAnalysis: "removed entries should disappear from list, search, and graph",
		AffectedFiles: []models.FileChange{
			{Path: "src/frontend/src/components/Dashboard.tsx", ChangeType: "modify"},
		},
		Tags: []string{"dashboard", "delete"},
	}

	if err := core.IndexEntry(entry); err != nil {
		t.Fatalf("IndexEntry failed: %v", err)
	}

	deleted, err := core.DeleteEntry(entry.ID)
	if err != nil {
		t.Fatalf("DeleteEntry failed: %v", err)
	}
	if deleted.ID != entry.ID || deleted.Summary != entry.Summary {
		t.Fatalf("DeleteEntry returned wrong entry: %#v", deleted)
	}

	entries, err := core.ListEntries(0, 10)
	if err != nil {
		t.Fatalf("ListEntries after delete failed: %v", err)
	}
	if containsEntry(entries, entry.ID) {
		t.Fatalf("expected deleted entry to be absent from list: %#v", entries)
	}

	results, err := core.SearchKnowledge("dashboard delete button", 10)
	if err != nil {
		t.Fatalf("SearchKnowledge after delete failed: %v", err)
	}
	for _, result := range results {
		if result.Entry.ID == entry.ID {
			t.Fatalf("expected deleted entry to be absent from search results: %#v", results)
		}
	}

	nodes, edges, err := core.GetGraphData(100)
	if err != nil {
		t.Fatalf("GetGraphData after delete failed: %v", err)
	}
	for _, node := range nodes {
		if node.ID == entry.ID {
			t.Fatalf("expected deleted entry node to be absent from graph: %#v", nodes)
		}
	}
	for _, edge := range edges {
		if edge.SourceID == entry.ID || edge.TargetID == entry.ID {
			t.Fatalf("expected deleted entry edges to be absent from graph: %#v", edges)
		}
	}

	if err := core.RestoreEntry(deleted); err != nil {
		t.Fatalf("RestoreEntry failed: %v", err)
	}
	entries, err = core.ListEntries(0, 10)
	if err != nil {
		t.Fatalf("ListEntries after restore failed: %v", err)
	}
	if !containsEntry(entries, entry.ID) {
		t.Fatalf("expected restored entry to return to list: %#v", entries)
	}

	results, err = core.SearchKnowledge("dashboard delete button", 10)
	if err != nil {
		t.Fatalf("SearchKnowledge after restore failed: %v", err)
	}
	found := false
	for _, result := range results {
		if result.Entry.ID == entry.ID {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected restored entry to return to search results: %#v", results)
	}

	if err := core.RestoreEntry(deleted); err != nil {
		t.Fatalf("second RestoreEntry failed: %v", err)
	}
	entries, err = core.ListEntries(0, 10)
	if err != nil {
		t.Fatalf("ListEntries after second restore failed: %v", err)
	}
	for _, listed := range entries {
		if listed.ID == entry.ID && len(listed.AffectedFiles) != len(entry.AffectedFiles) {
			t.Fatalf("expected repeated restore to keep %d affected files, got %d: %#v", len(entry.AffectedFiles), len(listed.AffectedFiles), listed.AffectedFiles)
		}
	}
}

func containsEntry(entries []models.StructuredEntry, id string) bool {
	for _, entry := range entries {
		if entry.ID == id {
			return true
		}
	}
	return false
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func gitRun(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, string(out))
	}
}
