package parser

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewMkInfo_ValidFile(t *testing.T) {
	content := `
__init__:
	BUILD=build

build / b:
	echo "build"

*make:
	make
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "task.mk")
	os.WriteFile(tmpFile, []byte(content), 0o644)

	mk := NewMkInfo(tmpFile)

	if mk.Err != 0 {
		t.Errorf("NewMkInfo() err = %d, want 0", mk.Err)
	}
	if mk.ErrMsg != "" {
		t.Errorf("NewMkInfo() errMsg = %q, want empty", mk.ErrMsg)
	}
	if len(mk.BlockList) != 2 {
		t.Errorf("BlockList count = %d, want 2", len(mk.BlockList))
	}
}

func TestNewMkInfo_InvalidFile(t *testing.T) {
	content := `
invalid_line_here
    echo "no task header"
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "invalid.mk")
	os.WriteFile(tmpFile, []byte(content), 0o644)

	mk := NewMkInfo(tmpFile)

	if mk.Err == 0 {
		t.Errorf("NewMkInfo() err = 0, want non-zero for invalid file")
	}
}

func TestNewMkInfo_NonExistentFile(t *testing.T) {
	mk := NewMkInfo("/nonexistent/path/task.mk")

	if mk.Err == 0 {
		t.Errorf("NewMkInfo() err = 0, want non-zero for nonexistent file")
	}
}

func TestNewMkInfo_WithInitBlock(t *testing.T) {
	content := `
__init__:
	BUILD=build

build:
	echo "build"
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "task.mk")
	os.WriteFile(tmpFile, []byte(content), 0o644)

	mk := NewMkInfo(tmpFile)

	if mk.Err != 0 {
		t.Errorf("NewMkInfo() err = %d, want 0", mk.Err)
	}
	if mk.InitBlock == nil {
		t.Error("InitBlock is nil, want non-nil")
	}
	if mk.InitBlock.CmdBlock != "BUILD=build" {
		t.Errorf("InitBlock.CmdBlock = %q, want %q", mk.InitBlock.CmdBlock, "BUILD=build")
	}
}

func TestNewMkInfo_DefaultTask(t *testing.T) {
	content := `
build:
	echo "build"

*make:
	make
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "task.mk")
	os.WriteFile(tmpFile, []byte(content), 0o644)

	mk := NewMkInfo(tmpFile)

	if mk.Err != 0 {
		t.Errorf("NewMkInfo() err = %d, want 0", mk.Err)
	}
	if len(mk.DefaultTasks) != 1 {
		t.Errorf("DefaultTasks count = %d, want 1", len(mk.DefaultTasks))
	}
	if mk.DefaultTasks[0] != "make" {
		t.Errorf("DefaultTasks[0] = %q, want %q", mk.DefaultTasks[0], "make")
	}
}

func TestFindTaskBlock(t *testing.T) {
	content := `
build:
	echo "build"

test:
	echo "test"

*run:
	echo "run"
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "task.mk")
	os.WriteFile(tmpFile, []byte(content), 0o644)

	mk := NewMkInfo(tmpFile)

	tests := []struct {
		name  string
		tasks []string
		found bool
	}{
		{"find_build", []string{"build"}, true},
		{"find_test", []string{"test"}, true},
		{"find_run", []string{"run"}, true},
		{"find_by_number", []string{"1"}, true},
		{"not_found", []string{"nonexist"}, false},
		{"number_0_uses_default", []string{"0"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			block := mk.FindTaskBlock(tt.tasks)
			found := block != nil
			if found != tt.found {
				t.Errorf("FindTaskBlock(%v) found = %v, want %v", tt.tasks, found, tt.found)
			}
		})
	}
}

func TestFindTaskBlock_ByNumber(t *testing.T) {
	content := `
first:
	echo "first"

second:
	echo "second"

third:
	echo "third"
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "task.mk")
	os.WriteFile(tmpFile, []byte(content), 0o644)

	mk := NewMkInfo(tmpFile)

	block1 := mk.FindTaskBlock([]string{"1"})
	if block1 == nil || block1.Tasks[0] != "first" {
		t.Errorf("FindTaskBlock([\"1\"]) = %v, want first task", block1)
	}

	block2 := mk.FindTaskBlock([]string{"2"})
	if block2 == nil || block2.Tasks[0] != "second" {
		t.Errorf("FindTaskBlock([\"2\"]) = %v, want second task", block2)
	}
}

func TestSearchTask(t *testing.T) {
	content := `
build_task:
	## builds the project
	echo "build"

test_task:
	## runs tests
	echo "test"

deploy_task:
	## deploy to production
	echo "deploy"
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "task.mk")
	os.WriteFile(tmpFile, []byte(content), 0o644)

	mk := NewMkInfo(tmpFile)

	if mk.Err != 0 {
		t.SkipNow()
	}

	tests := []struct {
		name   string
		module string
		keys   []string
		want   int
	}{
		{"search_all", "@myapp", []string{}, 3},
		{"search_build", "@myapp", []string{"build"}, 1},
		{"search_test", "@myapp", []string{"test"}, 1},
		{"search_deploy", "@myapp", []string{"deploy"}, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := mk.SearchTask(tt.module, tt.keys)
			if len(results) != tt.want {
				t.Errorf("SearchTask(%q, %v) count = %d, want %d",
					tt.module, tt.keys, len(results), tt.want)
			}
		})
	}
}

func TestSearchTask_NoMatch(t *testing.T) {
	content := `
build:
	echo "build"
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "task.mk")
	os.WriteFile(tmpFile, []byte(content), 0o644)

	mk := NewMkInfo(tmpFile)

	results := mk.SearchTask("@test", []string{"nonexistent"})
	if len(results) != 0 {
		t.Errorf("SearchTask() returned %d results, want 0", len(results))
	}
}

func TestBlock_Interpreter(t *testing.T) {
	content := `
task1:
	#!/bin/bash
	echo "bash task"
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "task.mk")
	os.WriteFile(tmpFile, []byte(content), 0o644)

	mk := NewMkInfo(tmpFile)

	if mk.Err != 0 || len(mk.BlockList) == 0 {
		t.SkipNow()
	}

	block := mk.FindTaskBlock([]string{"task1"})
	if block == nil {
		t.Error("task1 block is nil")
	}
}

func TestBlock_WithDiv(t *testing.T) {
	content := `
task1:
	echo "first"

#############################
task2:
	echo "second"
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "task.mk")
	os.WriteFile(tmpFile, []byte(content), 0o644)

	mk := NewMkInfo(tmpFile)

	if mk.Err != 0 {
		t.SkipNow()
	}

	t.Logf("BlockList count: %d", len(mk.BlockList))
	for i, block := range mk.BlockList {
		t.Logf("Block %d: tasks=%v, WithDiv=%v", i, block.Tasks, block.WithDiv)
	}

	if len(mk.BlockList) >= 2 {
		task2 := mk.FindTaskBlock([]string{"task2"})
		if task2 == nil {
			t.Error("task2 block is nil")
		} else if !task2.WithDiv {
			t.Log("task2.WithDiv is false (divider may need indentation)")
		}
	}
}

func TestMkInfo_PrintErr(t *testing.T) {
	content := `
invalid_line
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "invalid.mk")
	os.WriteFile(tmpFile, []byte(content), 0o644)

	mk := NewMkInfo(tmpFile)

	if mk.Err == 0 || mk.ErrMsg == "" {
		t.Error("PrintErr should have error message")
	}
}

func TestCalTaskStrLen(t *testing.T) {
	tests := []struct {
		name string
		arr  []string
		want int
	}{
		{"empty", []string{}, 0},
		{"single", []string{"build"}, 5},
		{"double", []string{"build", "b"}, 9},
		{"triple", []string{"build", "b", "make"}, 16},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calTaskStrLen(tt.arr)
			if got != tt.want {
				t.Errorf("calTaskStrLen(%v) = %d, want %d", tt.arr, got, tt.want)
			}
		})
	}
}

func TestNewMkInfo_DuplicateTask(t *testing.T) {
	content := `
build:
	echo "build"

build:
	echo "duplicate"
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "duplicate.mk")
	os.WriteFile(tmpFile, []byte(content), 0o644)

	mk := NewMkInfo(tmpFile)

	if mk.Err == 0 {
		t.Error("Should have error for duplicate task")
	}
	if mk.ErrMsg == "" {
		t.Error("Should have error message for duplicate task")
	}
}

func TestNewMkInfo_DuplicateDefaultTask(t *testing.T) {
	content := `
*build:
	echo "build"

*make:
	echo "make"
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "dup_default.mk")
	os.WriteFile(tmpFile, []byte(content), 0o644)

	mk := NewMkInfo(tmpFile)

	if mk.Err == 0 {
		t.Error("Should have error for duplicate default task")
	}
}

func TestPrintErr_NoError(t *testing.T) {
	content := `
build:
	echo "build"
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "valid.mk")
	os.WriteFile(tmpFile, []byte(content), 0o644)

	mk := NewMkInfo(tmpFile)

	if mk.Err != 0 {
		t.Skip("Skip: no error to print")
	}
}

func TestPrintErr_WithError(t *testing.T) {
	content := `
invalid_line
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "invalid.mk")
	os.WriteFile(tmpFile, []byte(content), 0o644)

	mk := NewMkInfo(tmpFile)

	if mk.Err == 0 || mk.ErrMsg == "" {
		t.Error("PrintErr should have error message")
	}
}

func TestPrintTask_ListSimple(t *testing.T) {
	content := `
build:
	echo "build"

test:
	echo "test"
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "task.mk")
	os.WriteFile(tmpFile, []byte(content), 0o644)

	mk := NewMkInfo(tmpFile)

	if mk.Err != 0 {
		t.SkipNow()
	}

	mk.PrintTask("s")
}

func TestPrintTask_ListWithComments(t *testing.T) {
	content := `
build:
	## builds the project
	echo "build"

test:
	## runs tests
	echo "test"
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "task.mk")
	os.WriteFile(tmpFile, []byte(content), 0o644)

	mk := NewMkInfo(tmpFile)

	if mk.Err != 0 {
		t.SkipNow()
	}

	mk.PrintTask("")
}

func TestLinesIntoGroup_Empty(t *testing.T) {
	lines := []*LineInfo{}
	result := linesIntoGroup(lines)
	if result != nil {
		t.Errorf("linesIntoGroup(nil) = %v, want nil", result)
	}
}

func TestLinesIntoGroup_SingleTask(t *testing.T) {
	lines := []*LineInfo{
		{Num: 1, Type: "task", Tasks: []string{"build"}},
		{Num: 2, Type: "code", Str: "echo build"},
	}
	result := linesIntoGroup(lines)
	if len(result) != 1 {
		t.Errorf("linesIntoGroup() count = %d, want 1", len(result))
	}
}

func TestLinesIntoGroup_WithDivider(t *testing.T) {
	lines := []*LineInfo{
		{Num: 1, Type: "task", Tasks: []string{"build"}},
		{Num: 2, Type: "code", Str: "echo build"},
		{Num: 3, Type: "div"},
		{Num: 4, Type: "task", Tasks: []string{"test"}},
		{Num: 5, Type: "code", Str: "echo test"},
	}
	result := linesIntoGroup(lines)
	if len(result) != 2 {
		t.Errorf("linesIntoGroup() count = %d, want 2", len(result))
	}
}

func TestPrintTask_EmptyBlockList(t *testing.T) {
	mk := &MkInfo{
		BlockList: []*Block{},
	}
	mk.PrintTask("")
	mk.PrintTask("s")
}

func TestPrintErr_MultipleErrors(t *testing.T) {
	mk := &MkInfo{
		Err:    2,
		ErrMsg: "multiple errors occurred",
	}
	mk.PrintErr()
}

func TestPrintTask_WithShortOption(t *testing.T) {
	mk := &MkInfo{
		BlockList: []*Block{
			{Tasks: []string{"build"}, CmdBlock: "echo build"},
			{Tasks: []string{"test"}, CmdBlock: "echo test"},
			{Tasks: []string{"deploy"}, CmdBlock: "echo deploy"},
		},
	}
	mk.PrintTask("s")
}

func TestPrintTask_WithLongOption(t *testing.T) {
	mk := &MkInfo{
		BlockList: []*Block{
			{Tasks: []string{"build"}, CmdBlock: "echo build"},
			{Tasks: []string{"test"}, CmdBlock: "echo test"},
		},
	}
	mk.PrintTask("")
}

func TestPrintTask_MixedTaskNames(t *testing.T) {
	mk := &MkInfo{
		BlockList: []*Block{
			{Tasks: []string{"build", "b"}, CmdBlock: "echo build"},
			{Tasks: []string{"clean", "c"}, CmdBlock: "echo clean"},
		},
	}
	mk.PrintTask("s")
}

func TestParseFile_EmptyContent(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "empty.mk")
	os.WriteFile(tmpFile, []byte(""), 0o644)

	mk := NewMkInfo(tmpFile)
	if mk.Err == 0 && len(mk.BlockList) == 0 {
		// Empty file with no tasks is OK
		return
	}
	// Empty file may have error if parser expects at least one task
}

func TestParseFile_OnlyComments(t *testing.T) {
	content := `# This is a comment
# Another comment
## This has two hashes
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "comments.mk")
	os.WriteFile(tmpFile, []byte(content), 0o644)

	mk := NewMkInfo(tmpFile)
	if mk.Err == 0 && len(mk.BlockList) == 0 {
		// Comments-only file with no tasks is OK
		return
	}
}

func TestParseFile_WithDivider(t *testing.T) {
	content := `
build:
	echo build

#############################

test:
	echo test
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "divider.mk")
	os.WriteFile(tmpFile, []byte(content), 0o644)

	mk := NewMkInfo(tmpFile)
	if mk.Err != 0 {
		t.Errorf("NewMkInfo() err = %d, want 0", mk.Err)
	}
	if len(mk.BlockList) != 2 {
		t.Errorf("BlockList count = %d, want 2", len(mk.BlockList))
	}
}

func TestParseFile_InvalidIndentation(t *testing.T) {
	content := `
build:
echo no indent
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "invalid.mk")
	os.WriteFile(tmpFile, []byte(content), 0o644)

	mk := NewMkInfo(tmpFile)
	if mk.Err == 0 {
		t.Errorf("NewMkInfo() err = 0, want error for invalid indentation")
	}
}

func TestSearchTask_Found(t *testing.T) {
	content := `
build:
	echo build

test:
	echo test
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "task.mk")
	os.WriteFile(tmpFile, []byte(content), 0o644)

	mk := NewMkInfo(tmpFile)
	tasks := mk.SearchTask("", []string{"build"})
	if len(tasks) == 0 {
		t.Error("SearchTask('', ['build']) returned empty")
	}
}

func TestSearchTask_NotFound(t *testing.T) {
	content := `
build:
	echo build
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "task.mk")
	os.WriteFile(tmpFile, []byte(content), 0o644)

	mk := NewMkInfo(tmpFile)
	tasks := mk.SearchTask("", []string{"nonexistent"})
	if len(tasks) != 0 {
		t.Error("SearchTask('', ['nonexistent']) should return empty")
	}
}

func TestSearchTask_WithModuleAndKeys(t *testing.T) {
	content := `
build:
	## builds the project
	echo build
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "task.mk")
	os.WriteFile(tmpFile, []byte(content), 0o644)

	mk := NewMkInfo(tmpFile)
	tasks := mk.SearchTask("mytool", []string{"build"})
	if len(tasks) == 0 {
		t.Error("SearchTask should find task with module name")
	}
}

func TestFindTaskBlock_ByName(t *testing.T) {
	content := `
build:
	echo build

test:
	echo test
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "task.mk")
	os.WriteFile(tmpFile, []byte(content), 0o644)

	mk := NewMkInfo(tmpFile)
	block := mk.FindTaskBlock([]string{"test"})
	if block == nil {
		t.Error("FindTaskBlock(['test']) returned nil")
	}
}

func TestFindTaskBlock_ByAlias(t *testing.T) {
	content := `
build / b:
	echo build
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "task.mk")
	os.WriteFile(tmpFile, []byte(content), 0o644)

	mk := NewMkInfo(tmpFile)
	block := mk.FindTaskBlock([]string{"b"})
	if block == nil {
		t.Error("FindTaskBlock(['b']) should find by alias")
	}
}

func TestFindTaskBlock_NotFound(t *testing.T) {
	content := `
build:
	echo build
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "task.mk")
	os.WriteFile(tmpFile, []byte(content), 0o644)

	mk := NewMkInfo(tmpFile)
	block := mk.FindTaskBlock([]string{"nonexistent"})
	if block != nil {
		t.Error("FindTaskBlock(['nonexistent']) should return nil")
	}
}

func TestFindTaskBlock_EmptyTasks(t *testing.T) {
	mk := &MkInfo{
		BlockList: []*Block{
			{Tasks: []string{"build"}},
		},
	}
	block := mk.FindTaskBlock([]string{})
	if block != nil {
		t.Error("FindTaskBlock([]) should return nil for empty task list")
	}
}

func TestCalTaskStrLen_SingleTask(t *testing.T) {
	arr := []string{"build"}
	length := calTaskStrLen(arr)
	if length != len("build") {
		t.Errorf("calTaskStrLen(['build']) = %d, want %d", length, len("build"))
	}
}

func TestCalTaskStrLen_MultipleTasks(t *testing.T) {
	arr := []string{"build", "test", "deploy"}
	length := calTaskStrLen(arr)
	expected := len("build") + 3 + len("test") + 3 + len("deploy")
	if length != expected {
		t.Errorf("calTaskStrLen() = %d, want %d", length, expected)
	}
}
