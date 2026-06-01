package memorycore

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
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
