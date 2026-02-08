package module

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSearchTasksInModule_EmptyDir(t *testing.T) {
	tmpDir := t.TempDir()
	repoDir := filepath.Join(tmpDir, "repo")
	os.MkdirAll(repoDir, 0o755)

	results := SearchTasksInModule(tmpDir, []string{})
	if results == nil {
		t.Error("SearchTasksInModule should return empty slice, not nil")
	}
	if len(results) != 0 {
		t.Errorf("SearchTasksInModule() count = %d, want 0", len(results))
	}
}

func TestSearchTasksInModule_WithFiles(t *testing.T) {
	tmpDir := t.TempDir()
	repoDir := filepath.Join(tmpDir, "repo")
	os.MkdirAll(repoDir, 0o755)

	content := `
build:
	## builds the project
	echo "build"

test:
	## runs tests
	echo "test"
`
	os.WriteFile(filepath.Join(repoDir, "mytool.mk"), []byte(content), 0o644)

	results := SearchTasksInModule(tmpDir, []string{})
	if len(results) == 0 {
		t.Error("SearchTasksInModule() should find tasks")
	}
}

func TestSearchTasksInModule_WithKeyword(t *testing.T) {
	tmpDir := t.TempDir()
	repoDir := filepath.Join(tmpDir, "repo")
	os.MkdirAll(repoDir, 0o755)

	content := `
build:
	## builds the project
	echo "build"

deploy:
	## deploy to production
	echo "deploy"
`
	os.WriteFile(filepath.Join(repoDir, "mytool.mk"), []byte(content), 0o644)

	results := SearchTasksInModule(tmpDir, []string{"build"})
	if len(results) != 1 {
		t.Errorf("SearchTasksInModule([build]) count = %d, want 1", len(results))
	}
}

func TestPrintRepo_InvalidDir(t *testing.T) {
	tmpDir := t.TempDir()

	// Should not panic, just log error
	PrintRepo(tmpDir, "@nonexistent", "")
}

func TestPrintRepo_ValidModule(t *testing.T) {
	tmpDir := t.TempDir()
	repoDir := filepath.Join(tmpDir, "repo", "mytool")
	os.MkdirAll(repoDir, 0o755)

	content := `
build:
	echo "build"
`
	os.WriteFile(filepath.Join(repoDir, "task.mk"), []byte(content), 0o755)

	PrintRepo(tmpDir, "@mytool/", "")
}

func TestSearchTasksInModule_NonExistentRepo(t *testing.T) {
	tmpDir := t.TempDir()
	nonExistentDir := filepath.Join(tmpDir, "nonexistent")

	results := SearchTasksInModule(nonExistentDir, []string{})
	if results == nil {
		t.Error("SearchTasksInModule should return empty slice, not nil")
	}
}

func TestPrintRepo_InvalidPattern(t *testing.T) {
	tmpDir := t.TempDir()

	// Should not panic
	PrintRepo(tmpDir, "@", "")
}

func TestPrintRepo_WithSubdir(t *testing.T) {
	tmpDir := t.TempDir()
	repoDir := filepath.Join(tmpDir, "repo", "mytool", "subdir")
	os.MkdirAll(repoDir, 0o755)

	content := `build: echo build`
	os.WriteFile(filepath.Join(repoDir, "task.mk"), []byte(content), 0o755)

	PrintRepo(tmpDir, "@mytool/subdir/", "")
}

func TestPrintRepo_ShortOption(t *testing.T) {
	tmpDir := t.TempDir()
	repoDir := filepath.Join(tmpDir, "repo", "mytool")
	os.MkdirAll(repoDir, 0o755)

	content := `build: echo build`
	os.WriteFile(filepath.Join(repoDir, "task.mk"), []byte(content), 0o755)

	PrintRepo(tmpDir, "@mytool/", "s")
}
