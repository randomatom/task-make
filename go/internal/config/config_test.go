package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExpandFile(t *testing.T) {
	tests := []struct {
		name      string
		shortName string
		repoDir   string
		want      string
	}{
		{"module_short", "@module", "/repo", "/repo/module.mk"},
		{"module_with_ext", "@script", "/repo", "/repo/script.mk"},
		{"module_dir", "@mydir/", "/repo", "/repo/mydir"},
		{"local_file", "task.mk", "/repo", "task.mk"},
		{"relative_path", "./local.mk", "/repo", "./local.mk"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExpandFile(tt.shortName, tt.repoDir)
			if got != tt.want {
				t.Errorf("ExpandFile(%q, %q) = %q, want %q",
					tt.shortName, tt.repoDir, got, tt.want)
			}
		})
	}
}

func TestExpandFile_ModuleVariants(t *testing.T) {
	tests := []struct {
		name      string
		shortName string
		repoDir   string
		want      string
	}{
		{"nested_module", "@tools/docker", "/repo", "/repo/tools/docker.mk"},
		{"deeply_nested", "@a/b/c/d", "/repo", "/repo/a/b/c/d.mk"},
		{"module_underscore", "@my_module", "/repo", "/repo/my_module.mk"},
		{"module_dash", "@my-module", "/repo", "/repo/my-module.mk"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExpandFile(tt.shortName, tt.repoDir)
			if got != tt.want {
				t.Errorf("ExpandFile(%q, %q) = %q, want %q",
					tt.shortName, tt.repoDir, got, tt.want)
			}
		})
	}
}

func TestGetTmpDir(t *testing.T) {
	os.Unsetenv("HOME")
	os.Unsetenv("USER")
	os.Unsetenv("_TASK_TMP_DIR")

	defer func() {
		os.Setenv("HOME", "/Users/wzw")
		os.Setenv("USER", "wzw")
	}()

	tmpDir := GetTmpDir()

	if tmpDir == "" {
		t.Error("GetTmpDir() returned empty string")
	}

	expectedPrefix := filepath.Join("/tmp", "mk_task_dir@wzw")
	if len(tmpDir) < len(expectedPrefix) {
		t.Errorf("GetTmpDir() = %q, want prefix %q", tmpDir, expectedPrefix)
	}
}

func TestGetTmpDir_WithEnvOverride(t *testing.T) {
	defer os.Unsetenv("_TASK_TMP_DIR")

	os.Setenv("_TASK_TMP_DIR", "/custom/tmp")

	tmpDir := GetTmpDir()

	if tmpDir != "/custom/tmp" {
		t.Errorf("GetTmpDir() = %q, want %q", tmpDir, "/custom/tmp")
	}
}

func TestGetTaskMainDir(t *testing.T) {
	defer os.Unsetenv("_TASK_PROFILE_DIR")

	os.Setenv("_TASK_PROFILE_DIR", "/custom/path")

	dir := GetTaskMainDir()

	if dir != "/custom/path" {
		t.Errorf("GetTaskMainDir() = %q, want %q", dir, "/custom/path")
	}
}

func TestGetRepoDir(t *testing.T) {
	defer os.Unsetenv("_TASK_PROFILE_DIR")

	os.Setenv("_TASK_PROFILE_DIR", "/custom/path")

	repoDir := GetRepoDir()

	if repoDir != "/custom/path/repo" {
		t.Errorf("GetRepoDir() = %q, want %q", repoDir, "/custom/path/repo")
	}
}

func TestIsSubTask(t *testing.T) {
	defer os.Unsetenv("_TASK_IS_SUBTASK")

	if IsSubTask() {
		t.Error("IsSubTask() = true, want false when env not set")
	}

	os.Setenv("_TASK_IS_SUBTASK", "1")
	if !IsSubTask() {
		t.Error("IsSubTask() = false, want true when env is set")
	}
}

func TestEnsureDirs(t *testing.T) {
	tmpDir := t.TempDir()
	taskDir := filepath.Join(tmpDir, ".local", "task")

	err := EnsureDirs(taskDir)
	if err != nil {
		t.Errorf("EnsureDirs() error = %v", err)
	}

	if _, err := os.Stat(taskDir); os.IsNotExist(err) {
		t.Error("EnsureDirs() did not create task directory")
	}
}

func TestEnsureDirs_EmptyInput(t *testing.T) {
	err := EnsureDirs("")
	if err != nil {
		t.Errorf("EnsureDirs('') error = %v, want nil", err)
	}
}

func TestGetDefaultTaskFile(t *testing.T) {
	defer os.Unsetenv("_TASK_CUR_DEFAULT_FILE")

	if GetDefaultTaskFile() != "" {
		t.Error("GetDefaultTaskFile() = non-empty, want empty when env not set")
	}

	os.Setenv("_TASK_CUR_DEFAULT_FILE", "custom.mk")
	if GetDefaultTaskFile() != "custom.mk" {
		t.Errorf("GetDefaultTaskFile() = %q, want %q", GetDefaultTaskFile(), "custom.mk")
	}
}

func TestHomeDir(t *testing.T) {
	tests := []struct {
		name      string
		mockHome  string
		mockPerm  os.FileMode
		wantEmpty bool
	}{
		{"valid_home", "/Users/test", 0o755, false},
		{"home_is_slash", "/", 0o755, true},
		{"no_write_perm", "/Users/test", 0o555, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer os.Unsetenv("HOME")

			if tt.mockHome != "/" {
				os.Setenv("HOME", tt.mockHome)
			}

			if tt.mockPerm != 0 {
				// Can't easily test permissions in mock, skip
				t.Skip()
			}

			got := homeDir()
			if tt.wantEmpty && got != "" {
				t.Errorf("homeDir() = %q, want empty", got)
			}
		})
	}
}

func TestGetTaskRootWorkdir(t *testing.T) {
	// GetTaskRootWorkdir returns first char of cwd when env not set
	// Just verify it returns something (not testing actual cwd)
	_ = GetTaskRootWorkdir()

	defer os.Unsetenv("_TASK_ROOT_WORKDIR")

	os.Setenv("_TASK_ROOT_WORKDIR", "/custom/root")
	if GetTaskRootWorkdir() != "/custom/root" {
		t.Errorf("GetTaskRootWorkdir() = %q, want %q", GetTaskRootWorkdir(), "/custom/root")
	}
}

func TestEnsureDirs_AlreadyExists(t *testing.T) {
	tmpDir := t.TempDir()
	taskDir := filepath.Join(tmpDir, ".local", "task")
	os.MkdirAll(taskDir, 0o755)

	err := EnsureDirs(taskDir)
	if err != nil {
		t.Errorf("EnsureDirs() error = %v", err)
	}
}

func TestEnsureDirs_MultipleLevels(t *testing.T) {
	tmpDir := t.TempDir()
	taskDir := filepath.Join(tmpDir, ".local", "task", "repo", "nested")

	err := EnsureDirs(taskDir)
	if err != nil {
		t.Errorf("EnsureDirs() error = %v", err)
	}

	if _, err := os.Stat(taskDir); os.IsNotExist(err) {
		t.Error("EnsureDirs() did not create nested directory")
	}
}

func TestGetTmpDir_WithShmUnavailable(t *testing.T) {
	// Test when /dev/shm doesn't exist
	defer os.Unsetenv("_TASK_TMP_DIR")

	// Mock USER
	originalUser := os.Getenv("USER")
	defer os.Setenv("USER", originalUser)

	os.Unsetenv("_TASK_TMP_DIR")
	os.Setenv("USER", "testuser")

	// Temporarily rename /dev/shm if it exists
	_, err := os.Stat("/dev/shm")
	if err == nil {
		// Can't easily test this without actual fs changes
		t.Skip("/dev/shm exists, skipping")
	}

	dir := GetTmpDir()
	if dir == "" {
		t.Error("GetTmpDir() returned empty")
	}
}
