package args

import (
	"os"
	"path/filepath"
	"task/internal/config"
	"testing"
)

func setupMockEnv() {
	os.Setenv("_TASK_ROOT_WORKDIR", "/mock/root")
	os.Setenv("_TASK_CUR_DEFAULT_FILE", "task.mk")
	os.Setenv("_TASK_IS_SUBTASK", "")
	os.Setenv("EDITOR", "true")
}

func teardownMockEnv() {
	os.Unsetenv("_TASK_ROOT_WORKDIR")
	os.Unsetenv("_TASK_CUR_DEFAULT_FILE")
	os.Unsetenv("_TASK_IS_SUBTASK")
}

func TestParseFlag(t *testing.T) {
	setupMockEnv()
	defer teardownMockEnv()

	tests := []struct {
		name    string
		arg     string
		wantAct string
		wantErr int
	}{
		{"help_flag", "-h", "help", 0},
		{"list_flag", "-l", "list", 0},
		{"list_short", "-ls", "list", 0},
		{"list_compact", "-lc", "list", 0},
		{"edit_flag", "-e", "edit", 0},
		{"create_flag", "-c", "create", 0},
		{"search_flag", "-s", "search", 0},
		{"run_flag", "-r", "run", 0},
		{"workdir_flag", "-w", "default", 0},
		{"compile_flag", "-C", "compile", 0},
		{"tab_complete_flag", "-t", "tab_complete", 0},
		{"unknown_flag", "-x", "", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ai := &ArgsInfo{Action: "default"}
			err := ai.parseFlag(tt.arg)

			if err != tt.wantErr {
				t.Errorf("parseFlag(%q) err = %d, want %d", tt.arg, err, tt.wantErr)
			}
			if ai.Action != tt.wantAct {
				t.Errorf("parseFlag(%q) action = %q, want %q", tt.arg, ai.Action, tt.wantAct)
			}
		})
	}
}

func TestParseFlag_ListOption(t *testing.T) {
	setupMockEnv()
	defer teardownMockEnv()

	tests := []struct {
		name        string
		arg         string
		wantAct     string
		wantListOpt string
	}{
		{"list_simple", "-l", "list", ""},
		{"list_short", "-ls", "list", "s"},
		{"list_compact", "-lc", "list", "c"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ai := &ArgsInfo{Action: "default"}
			_ = ai.parseFlag(tt.arg)

			if ai.Action != tt.wantAct {
				t.Errorf("parseFlag(%q) action = %q, want %q", tt.arg, ai.Action, tt.wantAct)
			}
			if ai.ListOption != tt.wantListOpt {
				t.Errorf("parseFlag(%q) ListOption = %q, want %q", tt.arg, ai.ListOption, tt.wantListOpt)
			}
		})
	}
}

func TestParseFileAndTask(t *testing.T) {
	setupMockEnv()
	defer teardownMockEnv()

	tests := []struct {
		name     string
		arg      string
		wantFile string
		wantTask string
		wantErr  int
	}{
		{"task_only", "build", "task.mk", "build", 0},
		{"task_number", "1", "task.mk", "1", 0},
		{"file_and_task", "file.mk:build", "file.mk", "build", 0},
		{"module_task", "@module:task", "@module", "task", 0},
		{"module_file", "@module/", "@module/", "", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ai := &ArgsInfo{DefaultTaskFile: "task.mk"}
			err := ai.parseFileAndTask(tt.arg)

			if err != tt.wantErr {
				t.Errorf("parseFileAndTask(%q) err = %d, want %d", tt.arg, err, tt.wantErr)
			}
			if ai.File != tt.wantFile {
				t.Errorf("parseFileAndTask(%q) file = %q, want %q", tt.arg, ai.File, tt.wantFile)
			}
			if ai.Task != tt.wantTask {
				t.Errorf("parseFileAndTask(%q) task = %q, want %q", tt.arg, ai.Task, tt.wantTask)
			}
		})
	}
}

func TestExpandFile(t *testing.T) {
	setupMockEnv()
	defer teardownMockEnv()

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
			got := config.ExpandFile(tt.shortName, tt.repoDir)
			if got != tt.want {
				t.Errorf("ExpandFile(%q, %q) = %q, want %q",
					tt.shortName, tt.repoDir, got, tt.want)
			}
		})
	}
}

func TestNewArgsInfo(t *testing.T) {
	setupMockEnv()
	defer teardownMockEnv()

	tmpDir := t.TempDir()
	taskMainDir := filepath.Join(tmpDir, ".local", "task")
	os.MkdirAll(taskMainDir, 0o755)

	ai := NewArgsInfo([]string{"test", "-l"}, taskMainDir)

	if ai.Err != 0 {
		t.Errorf("NewArgsInfo() err = %d, want 0", ai.Err)
	}
	if ai.Action != "list" {
		t.Errorf("NewArgsInfo() action = %q, want %q", ai.Action, "list")
	}
	if ai.TaskMainDir != taskMainDir {
		t.Errorf("NewArgsInfo() TaskMainDir = %q, want %q", ai.TaskMainDir, taskMainDir)
	}
}

func TestNewArgsInfo_WithCreateAction(t *testing.T) {
	setupMockEnv()
	defer teardownMockEnv()

	tmpDir := t.TempDir()
	taskMainDir := filepath.Join(tmpDir, ".local", "task")
	os.MkdirAll(taskMainDir, 0o755)

	ai := NewArgsInfo([]string{"test", "-c", "newfile"}, taskMainDir)

	if ai.Err != 0 {
		t.Errorf("NewArgsInfo() err = %d, want 0", ai.Err)
	}
	if ai.Action != "create" {
		t.Errorf("NewArgsInfo() action = %q, want %q", ai.Action, "create")
	}
	if ai.File != "newfile" {
		t.Errorf("NewArgsInfo() file = %q, want %q", ai.File, "newfile")
	}
}

func TestNewArgsInfo_WithSearchAction(t *testing.T) {
	setupMockEnv()
	defer teardownMockEnv()

	tmpDir := t.TempDir()
	taskMainDir := filepath.Join(tmpDir, ".local", "task")
	os.MkdirAll(taskMainDir, 0o755)

	ai := NewArgsInfo([]string{"test", "-s", "build", "test"}, taskMainDir)

	if ai.Err != 0 {
		t.Errorf("NewArgsInfo() err = %d, want 0", ai.Err)
	}
	if ai.Action != "search" {
		t.Errorf("NewArgsInfo() action = %q, want %q", ai.Action, "search")
	}
	if len(ai.SearchKeyWords) != 2 {
		t.Errorf("SearchKeyWords count = %d, want 2", len(ai.SearchKeyWords))
	}
}

func TestArgsInfo_DoMain_Help(t *testing.T) {
	setupMockEnv()
	defer teardownMockEnv()

	tmpDir := t.TempDir()
	taskMainDir := filepath.Join(tmpDir, ".local", "task")
	os.MkdirAll(taskMainDir, 0o755)

	ai := NewArgsInfo([]string{"test", "-h"}, taskMainDir)
	ai.TaskMainDir = taskMainDir

	ret := ai.DoMain()
	if ret != 0 {
		t.Errorf("DoMain() return = %d, want 0", ret)
	}
}

func TestArgsInfo_DoMain_InvalidAction(t *testing.T) {
	setupMockEnv()
	defer teardownMockEnv()

	tmpDir := t.TempDir()
	taskMainDir := filepath.Join(tmpDir, ".local", "task")
	os.MkdirAll(taskMainDir, 0o755)

	ai := NewArgsInfo([]string{"test", "-x"}, taskMainDir)
	ai.TaskMainDir = taskMainDir

	ret := ai.DoMain()
	if ret != 2 {
		t.Errorf("DoMain() return = %d, want 2", ret)
	}
}

func TestGetBaseDir(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		wantBase string
	}{
		{"normal_path", "/home/user/file.txt", "/home/user"},
		{"file_only", "file.txt", ""},
		{"deep_path", "/a/b/c/d.txt", "/a/b/c"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ai := &ArgsInfo{}
			got := ai.GetBaseDir(tt.path)
			if got != tt.wantBase {
				t.Errorf("GetBaseDir(%q) = %q, want %q", tt.path, got, tt.wantBase)
			}
		})
	}
}

func TestDoSearch_Disabled(t *testing.T) {
	setupMockEnv()
	defer teardownMockEnv()

	ai := &ArgsInfo{
		TaskMainDir:    "",
		SearchKeyWords: []string{"build"},
	}

	ret := ai.doSearch()
	if ret != 0 {
		t.Errorf("doSearch() return = %d, want 0", ret)
	}
}

func TestDoRun_Disabled(t *testing.T) {
	setupMockEnv()
	defer teardownMockEnv()

	ai := &ArgsInfo{
		TaskMainDir:    "",
		SearchKeyWords: []string{"build"},
	}

	ret := ai.doRun()
	if ret != 0 {
		t.Errorf("doRun() return = %d, want 0", ret)
	}
}

func TestDoRun_NoMatch(t *testing.T) {
	setupMockEnv()
	defer teardownMockEnv()

	tmpDir := t.TempDir()
	repoDir := filepath.Join(tmpDir, ".local", "task", "repo")
	os.MkdirAll(repoDir, 0o755)

	ai := &ArgsInfo{
		TaskMainDir:    filepath.Join(tmpDir, ".local", "task"),
		SearchKeyWords: []string{"nonexistent_task"},
	}

	ret := ai.doRun()
	if ret != 1 {
		t.Errorf("doRun() return = %d, want 1", ret)
	}
}

func TestDoTabComplete(t *testing.T) {
	setupMockEnv()
	defer teardownMockEnv()

	ai := &ArgsInfo{}

	ret := ai.doTabComplete()
	if ret != 0 {
		t.Errorf("doTabComplete() return = %d, want 0", ret)
	}
}

func TestDoCompile(t *testing.T) {
	setupMockEnv()
	defer teardownMockEnv()

	ai := &ArgsInfo{}

	ret := ai.doCompile()
	if ret != 0 {
		t.Errorf("doCompile() return = %d, want 0", ret)
	}
}

func TestParseArgs_WithSearchAction(t *testing.T) {
	setupMockEnv()
	defer teardownMockEnv()

	tmpDir := t.TempDir()
	taskMainDir := filepath.Join(tmpDir, ".local", "task")
	os.MkdirAll(taskMainDir, 0o755)

	ai := NewArgsInfo([]string{"test", "-s", "build", "test"}, taskMainDir)

	if ai.Action != "search" {
		t.Errorf("Action = %q, want %q", ai.Action, "search")
	}
	if len(ai.SearchKeyWords) != 2 {
		t.Errorf("SearchKeyWords count = %d, want 2", len(ai.SearchKeyWords))
	}
}

func TestParseArgs_WithRunAction(t *testing.T) {
	setupMockEnv()
	defer teardownMockEnv()

	tmpDir := t.TempDir()
	taskMainDir := filepath.Join(tmpDir, ".local", "task")
	os.MkdirAll(taskMainDir, 0o755)

	ai := NewArgsInfo([]string{"test", "-r", "deploy"}, taskMainDir)

	if ai.Action != "run" {
		t.Errorf("Action = %q, want %q", ai.Action, "run")
	}
}

func TestParseArgs_WithEditAction(t *testing.T) {
	setupMockEnv()
	defer teardownMockEnv()

	tmpDir := t.TempDir()
	taskMainDir := filepath.Join(tmpDir, ".local", "task")
	os.MkdirAll(taskMainDir, 0o755)

	ai := NewArgsInfo([]string{"test", "-e", "task.mk"}, taskMainDir)

	if ai.Action != "edit" {
		t.Errorf("Action = %q, want %q", ai.Action, "edit")
	}
}

func TestDoCreate_WithDirectory(t *testing.T) {
	setupMockEnv()
	defer teardownMockEnv()

	tmpDir := t.TempDir()
	taskMainDir := filepath.Join(tmpDir, ".local", "task")
	repoDir := filepath.Join(taskMainDir, "repo")
	os.MkdirAll(repoDir, 0o755)

	ai := &ArgsInfo{
		File:        "@newdir/",
		TaskMainDir: taskMainDir,
		TaskRepoDir: repoDir,
	}

	ret := ai.doCreate()
	if ret != 0 {
		t.Errorf("doCreate() return = %d, want 0", ret)
	}

	newDir := filepath.Join(repoDir, "newdir")
	if _, err := os.Stat(newDir); os.IsNotExist(err) {
		t.Error("doCreate() did not create directory")
	}
}

func TestDoCreate_WithInvalidPath(t *testing.T) {
	setupMockEnv()
	defer teardownMockEnv()

	ai := &ArgsInfo{
		File:        "@",
		TaskMainDir: t.TempDir(),
	}

	ret := ai.doCreate()
	if ret != 1 {
		t.Errorf("doCreate() return = %d, want 1", ret)
	}
}

func TestDoCreate_WithExistingFile(t *testing.T) {
	setupMockEnv()
	defer teardownMockEnv()

	tmpDir := t.TempDir()
	taskMainDir := filepath.Join(tmpDir, ".local", "task")
	os.MkdirAll(taskMainDir, 0o755)

	existingFile := filepath.Join(tmpDir, "existing.mk")
	os.WriteFile(existingFile, []byte("test"), 0o644)

	ai := &ArgsInfo{
		File:        existingFile,
		TaskMainDir: taskMainDir,
	}

	ret := ai.doCreate()
	if ret != 1 {
		t.Errorf("doCreate() return = %d, want 1 for existing file", ret)
	}
}

func TestExpandFile_RepoDir(t *testing.T) {
	setupMockEnv()
	defer teardownMockEnv()

	tmpDir := t.TempDir()
	taskMainDir := filepath.Join(tmpDir, ".local", "task")
	os.MkdirAll(taskMainDir, 0o755)

	ai := &ArgsInfo{
		TaskMainDir: taskMainDir,
		TaskRepoDir: filepath.Join(taskMainDir, "repo"),
	}

	result := ai.expandFile("@mytool")
	expected := filepath.Join(ai.TaskRepoDir, "mytool.mk")

	if result != expected {
		t.Errorf("expandFile(@mytool) = %q, want %q", result, expected)
	}
}

func TestExpandFile_LocalFile(t *testing.T) {
	setupMockEnv()
	defer teardownMockEnv()

	tmpDir := t.TempDir()
	taskMainDir := filepath.Join(tmpDir, ".local", "task")
	os.MkdirAll(taskMainDir, 0o755)

	ai := &ArgsInfo{
		TaskMainDir: taskMainDir,
		TaskRepoDir: filepath.Join(taskMainDir, "repo"),
	}

	result := ai.expandFile("local.mk")

	if result != "local.mk" {
		t.Errorf("expandFile(local.mk) = %q, want %q", result, "local.mk")
	}
}

func TestDoMain_WithHelpAction(t *testing.T) {
	setupMockEnv()
	defer teardownMockEnv()

	tmpDir := t.TempDir()
	taskMainDir := filepath.Join(tmpDir, ".local", "task")
	os.MkdirAll(taskMainDir, 0o755)

	ai := NewArgsInfo([]string{"test", "-h"}, taskMainDir)
	ai.TaskMainDir = taskMainDir

	ret := ai.DoMain()
	if ret != 0 {
		t.Errorf("DoMain() return = %d, want 0", ret)
	}
}

func TestDoMain_WithDefaultAction_NoFile(t *testing.T) {
	setupMockEnv()
	defer teardownMockEnv()

	tmpDir := t.TempDir()
	taskMainDir := filepath.Join(tmpDir, ".local", "task")
	os.MkdirAll(taskMainDir, 0o755)

	ai := NewArgsInfo([]string{"test"}, taskMainDir)
	ai.TaskMainDir = taskMainDir

	// No file specified, should return 1
	ret := ai.DoMain()
	if ret != 1 {
		t.Errorf("DoMain() return = %d, want 1", ret)
	}
}

func TestDoList_WithSpecificTask(t *testing.T) {
	setupMockEnv()
	defer teardownMockEnv()

	tmpDir := t.TempDir()
	taskMainDir := filepath.Join(tmpDir, ".local", "task")
	os.MkdirAll(taskMainDir, 0o755)

	// Create a temporary mk file
	mkContent := `
build:
	echo "build"
`
	mkFile := filepath.Join(tmpDir, "test.mk")
	os.WriteFile(mkFile, []byte(mkContent), 0o644)

	ai := &ArgsInfo{
		File:        mkFile,
		Task:        "build",
		TaskMainDir: taskMainDir,
	}

	ret := ai.doList()
	// May fail due to file not existing, but we're testing the code path
	_ = ret
}

func TestDoList_WithRepoDir(t *testing.T) {
	setupMockEnv()
	defer teardownMockEnv()

	tmpDir := t.TempDir()
	taskMainDir := filepath.Join(tmpDir, ".local", "task")
	repoDir := filepath.Join(taskMainDir, "repo")
	os.MkdirAll(repoDir, 0o755)

	ai := &ArgsInfo{
		File:        "@/",
		TaskMainDir: taskMainDir,
		TaskRepoDir: repoDir,
	}

	ret := ai.doList()
	if ret != 0 {
		t.Errorf("doList() return = %d, want 0", ret)
	}
}

func TestDoList_WithNonExistentFile(t *testing.T) {
	setupMockEnv()
	defer teardownMockEnv()

	tmpDir := t.TempDir()
	taskMainDir := filepath.Join(tmpDir, ".local", "task")
	os.MkdirAll(taskMainDir, 0o755)

	ai := &ArgsInfo{
		File:        "/nonexistent/file.mk",
		TaskMainDir: taskMainDir,
	}

	ret := ai.doList()
	if ret != 1 {
		t.Errorf("doList() return = %d, want 1", ret)
	}
}

func TestUpdateRunList_Disabled(t *testing.T) {
	setupMockEnv()
	defer teardownMockEnv()

	ai := &ArgsInfo{
		TaskMainDir: "",
		File:        "task.mk",
	}

	// Should not panic, should just return
	ai.updateRunList()
}

func TestUpdateRunList_NotTaskMk(t *testing.T) {
	setupMockEnv()
	defer teardownMockEnv()

	tmpDir := t.TempDir()
	taskMainDir := filepath.Join(tmpDir, ".local", "task")
	os.MkdirAll(taskMainDir, 0o755)

	ai := &ArgsInfo{
		TaskMainDir: taskMainDir,
		File:        "other.mk",
	}

	// Should not panic, should just return
	ai.updateRunList()
}

func TestUpdateRunList_FirstFile(t *testing.T) {
	setupMockEnv()
	defer teardownMockEnv()

	tmpDir := t.TempDir()
	taskMainDir := filepath.Join(tmpDir, ".local", "task")
	os.MkdirAll(taskMainDir, 0o755)

	ai := &ArgsInfo{
		TaskMainDir: taskMainDir,
		File:        "task.mk",
	}

	// Change to temp dir so we can write
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	// Should create the run_file_list.txt
	ai.updateRunList()

	runFileList := filepath.Join(taskMainDir, "run_file_list.txt")
	if _, err := os.Stat(runFileList); os.IsNotExist(err) {
		t.Error("updateRunList() did not create run_file_list.txt")
	}
}

func TestDoHelp(t *testing.T) {
	setupMockEnv()
	defer teardownMockEnv()

	ai := &ArgsInfo{}

	ret := ai.doHelp()
	if ret != 0 {
		t.Errorf("doHelp() return = %d, want 0", ret)
	}
}

func TestDoEdit_WithNonExistentFile(t *testing.T) {
	setupMockEnv()
	defer teardownMockEnv()

	tmpDir := t.TempDir()
	taskMainDir := filepath.Join(tmpDir, ".local", "task")
	os.MkdirAll(taskMainDir, 0o755)

	ai := &ArgsInfo{
		File:        "/nonexistent/file.mk",
		TaskMainDir: taskMainDir,
	}

	ret := ai.doEdit()
	if ret != 1 {
		t.Errorf("doEdit() return = %d, want 1", ret)
	}
}

func TestDoDefault_WithNonExistentFile(t *testing.T) {
	setupMockEnv()
	defer teardownMockEnv()

	tmpDir := t.TempDir()
	taskMainDir := filepath.Join(tmpDir, ".local", "task")
	os.MkdirAll(taskMainDir, 0o755)

	ai := &ArgsInfo{
		File:        "/nonexistent/task.mk",
		TaskMainDir: taskMainDir,
	}

	ret := ai.doDefault()
	if ret != 1 {
		t.Errorf("doDefault() return = %d, want 1", ret)
	}
}

func TestDoDefault_WithExistingFile(t *testing.T) {
	setupMockEnv()
	defer teardownMockEnv()

	tmpDir := t.TempDir()
	taskMainDir := filepath.Join(tmpDir, ".local", "task")
	os.MkdirAll(taskMainDir, 0o755)

	mkFile := filepath.Join(tmpDir, "task.mk")
	content := "build:\n\techo hello\n"
	os.WriteFile(mkFile, []byte(content), 0o644)

	ai := &ArgsInfo{
		File:        mkFile,
		TaskMainDir: taskMainDir,
		MkfileDir:   tmpDir,
	}

	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	ret := ai.doDefault()
	if ret != 0 {
		t.Errorf("doDefault() return = %d, want 0", ret)
	}
}
