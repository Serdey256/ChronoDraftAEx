package depsdetect

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestParseGoMod(t *testing.T) {
	dir := t.TempDir()
	content := `module example.com/mymod

go 1.21

require (
	github.com/gin-gonic/gin v1.9.1
	github.com/spf13/cobra v1.8.0
	golang.org/x/text v0.14.0 // indirect
)
`
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	got, err := DetectDependencies(dir)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"github.com/gin-gonic/gin@v1.9.1",
		"github.com/spf13/cobra@v1.8.0",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestParsePackageJson(t *testing.T) {
	dir := t.TempDir()
	content := `{
  "name": "my-app",
  "dependencies": {
    "react": "^18.2.0",
    "react-dom": "^18.2.0"
  },
  "devDependencies": {
    "typescript": "^5.3.0"
  }
}
`
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	got, err := DetectDependencies(dir)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"react-dom@^18.2.0",
		"react@^18.2.0",
		"typescript@^5.3.0",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestParseRequirementsTxt(t *testing.T) {
	dir := t.TempDir()
	content := `# This is a comment
flask==2.3.0
requests>=2.28.0
--index-url https://pypi.org/simple
numpy
`
	if err := os.WriteFile(filepath.Join(dir, "requirements.txt"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	got, err := DetectDependencies(dir)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"flask@2.3.0",
		"numpy",
		"requests@2.28.0",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestNoDepFiles(t *testing.T) {
	dir := t.TempDir()
	got, err := DetectDependencies(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty slice, got %v", got)
	}
}
