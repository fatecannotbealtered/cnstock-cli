package lint

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// findProjectRoot walks up from the test file directory to find go.mod.
func findProjectRoot() string {
	_, src, _, _ := runtime.Caller(0)
	dir := filepath.Dir(src)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	wd, _ := os.Getwd()
	return wd
}

// goFiles returns all .go files under root, skipping hidden directories
// (.claude, .git, .idea, etc.) and vendor/node_modules.
func goFiles(root string) ([]string, error) {
	var files []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			if strings.HasSuffix(path, ".go") {
				files = append(files, path)
			}
			return nil
		}
		name := info.Name()
		if strings.HasPrefix(name, ".") || name == "vendor" || name == "node_modules" {
			return filepath.SkipDir
		}
		return nil
	})
	return files, err
}

func TestGofmt(t *testing.T) {
	root := findProjectRoot()

	files, err := goFiles(root)
	if err != nil {
		t.Fatalf("failed to collect .go files: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("no .go files found")
	}

	cmd := exec.Command("gofmt", append([]string{"-l"}, files...)...)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("gofmt failed: %v", err)
	}

	unformatted := strings.TrimSpace(string(out))
	if unformatted != "" {
		lines := strings.Split(unformatted, "\n")
		msg := fmt.Sprintf("the following %d file(s) need 'gofmt -w':\n", len(lines))
		for _, l := range lines {
			rel, _ := filepath.Rel(root, l)
			msg += "  " + rel + "\n"
		}
		t.Error(msg)
	}
}

func TestGoVet(t *testing.T) {
	root := findProjectRoot()

	cmd := exec.Command("go", "vet", "./...")
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go vet failed:\n%s\n%v", string(out), err)
	}
}
