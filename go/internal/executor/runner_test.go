package executor

import (
	"os"
	"path/filepath"
	"task/internal/parser"
	"testing"
)

func TestNewRunner(t *testing.T) {
	runner := NewRunner("/test/home", "/tmp/test")
	if runner.TaskMainDir != "/test/home" {
		t.Errorf("NewRunner() TaskMainDir = %q, want %q", runner.TaskMainDir, "/test/home")
	}
	if runner.TmpDir != "/tmp/test" {
		t.Errorf("NewRunner() TmpDir = %q, want %q", runner.TmpDir, "/tmp/test")
	}
}

func TestFindInPath(t *testing.T) {
	runner := &Runner{}

	// Test finding existing command
	shPath := runner.findInPath("sh")
	if shPath == "" {
		t.Error("findInPath('sh') should find shell path")
	}

	// Test non-existent command
	nonExistent := runner.findInPath("thiscommanddoesnotexist12345")
	if nonExistent != "" {
		t.Errorf("findInPath() should return empty for non-existent command")
	}
}

func TestWriteScript(t *testing.T) {
	runner := &Runner{TmpDir: t.TempDir()}

	content := "echo 'test'"
	scriptPath := filepath.Join(runner.TmpDir, "test_script.sh")

	runner.writeScript(scriptPath, content)

	// Check file exists and has correct content
	data, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Errorf("ReadFile() error = %v", err)
	}
	if string(data) != content {
		t.Errorf("writeScript() content = %q, want %q", string(data), content)
	}
}

func TestWriteScript_NoOverwrite(t *testing.T) {
	runner := &Runner{TmpDir: t.TempDir()}

	content1 := "echo 'first'"
	scriptPath := filepath.Join(runner.TmpDir, "test_script.sh")

	runner.writeScript(scriptPath, content1)

	runner.writeScript(scriptPath, content1)

	data, _ := os.ReadFile(scriptPath)
	if string(data) != content1 {
		t.Error("writeScript() should not overwrite if content is same")
	}
}

func TestWriteScript_OverwriteDifferent(t *testing.T) {
	runner := &Runner{TmpDir: t.TempDir()}

	content1 := "echo 'first'"
	content2 := "echo 'second'"
	scriptPath := filepath.Join(runner.TmpDir, "test_script.sh")

	runner.writeScript(scriptPath, content1)

	runner.writeScript(scriptPath, content2)

	data, _ := os.ReadFile(scriptPath)
	if string(data) != content2 {
		t.Errorf("writeScript() should overwrite if content is different, got %q, want %q", string(data), content2)
	}
}

func TestRunTask_InvalidFile(t *testing.T) {
	runner := &Runner{
		TaskMainDir: t.TempDir(),
		TmpDir:      t.TempDir(),
	}

	block := &parser.Block{
		Tasks:    []string{"test"},
		CmdBlock: "echo test",
	}

	ret := runner.RunTask("/nonexistent/file.mk", block, nil, nil, "", "")
	if ret == 0 {
		t.Error("RunTask() should return error for invalid file")
	}
}

func TestNewRunner_IsSubTask(t *testing.T) {
	// Test that NewRunner sets IsSubTask based on environment
	os.Setenv("_TASK_IS_SUBTASK", "1")
	defer os.Unsetenv("_TASK_IS_SUBTASK")

	runner := NewRunner("/test/home", "/tmp/test")
	if !runner.IsSubTask {
		t.Error("NewRunner() IsSubTask should be true when env is set")
	}
}

func TestNewRunner_ScriptArgs(t *testing.T) {
	// Save original os.Args
	origArgs := os.Args
	os.Args = []string{"m", "arg1", "arg2"}
	defer func() { os.Args = origArgs }()

	runner := NewRunner("/test/home", "/tmp/test")
	if len(runner.ScriptArgs) != 3 {
		t.Errorf("ScriptArgs count = %d, want 3", len(runner.ScriptArgs))
	}
}

func TestFindInPath_WithPath(t *testing.T) {
	runner := &Runner{}

	// Test finding a command in PATH
	shPath := runner.findInPath("sh")
	if shPath == "" {
		t.Error("findInPath('sh') should find shell path")
	}

	// Verify it's a valid path
	if !filepath.IsAbs(shPath) {
		t.Errorf("findInPath() returned relative path: %q", shPath)
	}
}

func TestRunTask_InvalidInterpreter(t *testing.T) {
	runner := &Runner{
		TaskMainDir: t.TempDir(),
		TmpDir:      t.TempDir(),
	}

	block := &parser.Block{
		Tasks:       []string{"build"},
		CmdBlock:    "echo test",
		Interpreter: "nonexistent_interpreter_12345",
	}

	ret := runner.RunTask("/nonexistent/file.mk", block, nil, nil, "", "")
	if ret == 0 {
		t.Error("RunTask() should return error for invalid interpreter")
	}
}
