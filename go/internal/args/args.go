package args

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"task/internal/config"
	"task/internal/executor"
	"task/internal/logger"
	"task/internal/module"
	"task/internal/parser"
)

type ArgsInfo struct {
	Action          string
	File            string
	CompileBashFile string
	Task            string
	ScriptArgs      []string
	ListOption      string
	SearchOption    string
	SearchKeyWords  []string
	ShellArgs       []string
	IncompleteText  string
	TaskMainDir     string
	TaskRepoDir     string
	TaskRootWorkdir string
	ChangeWorkdir   bool
	MkfileDir       string
	DefaultTaskFile string
	IsSubTask       bool
	TmpDir          string
	Err             int
}

func NewArgsInfo(scriptArgs []string, taskMainDir string) *ArgsInfo {
	ai := &ArgsInfo{
		Action:          "default",
		File:            "",
		CompileBashFile: "",
		Task:            "",
		ScriptArgs:      scriptArgs,
		ListOption:      "",
		SearchOption:    "",
		SearchKeyWords:  []string{},
		ShellArgs:       []string{},
		IncompleteText:  "",
		TaskMainDir:     taskMainDir,
		TaskRepoDir:     filepath.Join(taskMainDir, "repo"),
		TaskRootWorkdir: config.GetTaskRootWorkdir(),
		ChangeWorkdir:   false,
		MkfileDir:       "",
		DefaultTaskFile: config.GetDefaultTaskFile(),
		IsSubTask:       config.IsSubTask(),
		Err:             0,
	}

	if ai.DefaultTaskFile == "" {
		ai.DefaultTaskFile = "task.mk"
	}

	ai.TmpDir = config.GetTmpDir()
	if ai.TmpDir == "" {
		ai.Err = 1
		logger.Loge("Failed to create tmp dir")
		return ai
	}

	if ai.Err == 0 {
		ai.Err = ai.parseArgs()
	}

	return ai
}

func (ai *ArgsInfo) parseArgs() int {
	ai.File = ai.DefaultTaskFile
	i := 1

	for i < len(ai.ScriptArgs) {
		arg := ai.ScriptArgs[i]
		if len(arg) > 0 && arg[0] == '-' {
			ret := ai.parseFlag(arg)
			if ret != 0 {
				logger.Logf("option [%s] error", arg)
				return ret
			}
		} else {
			break
		}
		i++
	}

	if i < len(ai.ScriptArgs) {
		if ai.Action == "list" || ai.Action == "default" {
			if ai.parseFileAndTask(ai.ScriptArgs[i]) == 0 {
				ai.ShellArgs = ai.ScriptArgs[i+1:]
			} else {
				return 1
			}
		} else if ai.Action == "create" {
			ai.File = ai.ScriptArgs[i]
		} else if ai.Action == "tab_complete" {
			ai.IncompleteText = ai.ScriptArgs[i]
		} else if ai.Action == "edit" {
			ai.parseFileAndTask(ai.ScriptArgs[i])
		} else if ai.Action == "search" || ai.Action == "run" {
			ai.SearchKeyWords = ai.ScriptArgs[i:]
		} else if ai.Action == "compile" {
			ai.File = ai.ScriptArgs[i]
			if i+1 < len(ai.ScriptArgs) {
				ai.CompileBashFile = ai.ScriptArgs[i+1]
			} else {
				re := regexp.MustCompile(`\.mk$`)
				ai.CompileBashFile = re.ReplaceAllString(ai.File, ".sh")
			}
		} else {
			return 1
		}
	}

	return 0
}

func (ai *ArgsInfo) parseFlag(arg string) int {
	if len(arg) == 0 || arg[0] != '-' {
		return 1
	}

	switch arg[1] {
	case 'l':
		ai.Action = "list"
		if len(arg) > 2 && (arg[2] == 's' || arg[2] == 'c') {
			ai.ListOption = string(arg[2])
		}
	case 'e':
		ai.Action = "edit"
	case 'c':
		ai.Action = "create"
	case 's':
		ai.Action = "search"
	case 'r':
		ai.Action = "run"
	case 'w':
		ai.ChangeWorkdir = true
	case 'C':
		ai.Action = "compile"
	case 't':
		ai.Action = "tab_complete"
	case 'h':
		ai.Action = "help"
	default:
		ai.Action = ""
		return 1
	}

	return 0
}

func (ai *ArgsInfo) parseFileAndTask(arg string) int {
	ai.File = ""
	ai.Task = ""
	arr := strings.SplitN(arg, ":", 2)

	if len(arr) == 2 && (arr[0] != "" || arr[1] != "") {
		mFile, _ := regexp.MatchString(`^@\w+(\/|\w|\.|-)*$|^(\/|\w|\.|-)+$|^$`, arr[0])
		mTask, _ := regexp.MatchString(`^[A-Za-z_][\w|-]*$|^\d+$|^$`, arr[1])
		if mFile && mTask {
			if arr[0] != "" {
				ai.File = arr[0]
			}
			ai.Task = arr[1]
			return 0
		}
	} else if len(arr) == 1 {
		mFile, _ := regexp.MatchString(`^@(\/|\w|\.|-)*$`, arr[0])
		if mFile {
			ai.File = arr[0]
			ai.Task = ""
			return 0
		}

		if strings.HasSuffix(arr[0], ".mk") {
			ai.File = arr[0]
			ai.Task = ""
			return 0
		}

		mTask, _ := regexp.MatchString(`^[A-Za-z_][\w-]*$|^\d+$|^$`, arr[0])
		if mTask {
			ai.File = ai.DefaultTaskFile
			ai.Task = arr[0]
			return 0
		}
	}

	logger.Logf("Task [%s] Illegal format!", arg)
	if strings.HasSuffix(arg, ".mk") {
		if fi, err := os.Stat(arg); err == nil && !fi.IsDir() {
			logger.Logf("    Do you mean \"%s:\"?", arg)
		}
	}
	return 1
}

func (ai *ArgsInfo) expandFile(shortName string) string {
	return config.ExpandFile(shortName, ai.TaskRepoDir)
}

func (ai *ArgsInfo) DoMain() int {
	switch ai.Action {
	case "help":
		return ai.doHelp()
	case "create":
		return ai.doCreate()
	case "edit":
		return ai.doEdit()
	case "search":
		return ai.doSearch()
	case "run":
		return ai.doRun()
	case "list":
		return ai.doList()
	case "compile":
		if ai.File == "" {
			ai.File = "task.mk"
		}
		if ai.CompileBashFile == "" {
			ai.CompileBashFile = "task.sh"
		}
		return ai.doCompile()
	case "tab_complete":
		return ai.doTabComplete()
	case "default":
		return ai.doDefault()
	default:
		logger.Loge("Option error")
		return 2
	}
}

func (ai *ArgsInfo) doHelp() int {
	logger.Log("Usage:")
	logger.Log("    m    [[file:]task] [arg]...")
	logger.Log("    m -w [file:task]   [arg]...")
	logger.Log("    m -c [file]")
	logger.Log("    m -l [file]")
	logger.Log("    m -e [file]")
	logger.Log("    m -s [pattern]...")
	logger.Log("    m -r [pattern]...")
	logger.Log("    m -C [mk_file] [sh_file]")
	logger.Log("    -h")
	logger.Log("Arguments:")
	logger.Log("    [[file:]task]    Specifies the task name of the file. If not specified, the default")
	logger.Log("                     is the task.mk file in the current directory.")
	logger.Log("    [pattern]        the search pattern.")
	logger.Log("Options:")
	logger.Log("    -w               Change to WORKDIR where the task file is located before executing the task.")
	logger.Log("    -c               Create the mk_file.")
	logger.Log("    -e               Edit the mk_file.")
	logger.Log("    -l               List the tasks.")
	logger.Log("    -s               Search the tasks in global module.")
	logger.Log("    -r               Run the tasks in global module.")
	logger.Log("    -C               Compile the *.mk file into *.sh.")
	logger.Log("    -h               Help.")
	logger.Log("")
	logger.Log("Note: Global module features (-s, -r, @module) require home directory access.")
	logger.Log("      On Android, only local task.mk files are supported.")
	logger.Log("")
	logger.Log("Example template for task.mk script:")
	logger.Log("")
	logger.Log("__init__:")
	logger.Log("    # 该模块在所有后续任务执行前调用，用于设定公共内容")
	logger.Log("    BUILD=build")
	logger.Log("")
	logger.Log("build / b:")
	logger.Log("    # 内部注释，该行不会被 -l 显示。（前面只有一个#)")
	logger.Log("    # 上面的\"/\" 后面的 mp是简称，方便输入.")
	logger.Log("    rm -rf ${BUILD}")
	logger.Log("    mkdir ${BUILD}")
	logger.Log("    cd ${BUILD}")
	logger.Log("    cmake ..")
	logger.Log("")
	logger.Log("*make:")
	logger.Log("    # [*]代表默认任务. 当执行 m, 后面没有参数，直接执行该任务")
	logger.Log("    cd ${BUILD}")
	logger.Log("    make -j8")
	logger.Log("")
	logger.Log("claen:")
	logger.Log("    ## 任务后面第一行开头有两个##, 该行会被 -l 显示")
	logger.Log("    cd ${BUILD}")
	logger.Log("    make clean")
	logger.Log("")
	logger.Log("MK_DIR_exam:")
	logger.Log("    # workdir默认由顶层task.mk设定，使子脚本无法识别自身路径，难以引用当前目录文件")
	logger.Log("    # 新增 MK_DIR 变量，标识mk脚本所在目录")
	logger.Log("    python ${MK_DIR}/run.py")
	logger.Log("")
	logger.Log("#############################")
	logger.Log("######  会显示分隔线 ########")
	logger.Log("")
	logger.Log("all:")
	logger.Log("    # 可以用 m 调用其他任务")
	logger.Log("    m build")
	logger.Log("    m make")
	return 0
}

func (ai *ArgsInfo) doCreate() int {
	if ai.File == "" || ai.File == "@" {
		return 1
	}

	file := ai.expandFile(ai.File)

	if strings.HasSuffix(file, "/") {
		if err := os.MkdirAll(file, 0o755); err != nil {
			logger.Logf("Create Directory [%s] Error!", file)
			return 1
		}
	} else {
		if _, err := os.Stat(file); err == nil {
			logger.Logf("%s has exist!", file)
			return 1
		}

		f, err := os.Create(file)
		if err != nil {
			logger.Logf("create [%s] Error!", file)
			return 1
		}
		defer f.Close()

		f.WriteString("__init__:\n    # echo ${MK_DIR}\n\n*run1 / r:\n    pwd\n")
		logger.Logi2("create " + file)
	}
	return 0
}

func (ai *ArgsInfo) doEdit() int {
	file := ai.expandFile(ai.File)
	if _, err := os.Stat(file); os.IsNotExist(err) {
		logger.Logf("%s DON'T exist!", ai.File)
		return 1
	}

	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}

	if ai.Task != "" {
		info := parser.NewMkInfo(file)
		if info.Err == 0 {
			block := info.FindTaskBlock([]string{ai.Task})
			if block == nil {
				f, _ := os.OpenFile(file, os.O_APPEND|os.O_WRONLY, 0o644)
				if f != nil {
					f.WriteString("\n" + ai.Task + ":\n")
					f.Close()
				}
				runEditor(editor, file)
			} else {
				runEditor(editor, file, block.StartLineNum)
			}
		} else {
			logger.Loge(info.ErrMsg)
			runEditor(editor, file)
		}
	} else {
		runEditor(editor, file)
	}
	return 0
}

func runEditor(editor, file string, lineNum ...int) {
	var cmd *exec.Cmd
	if len(lineNum) > 0 && lineNum[0] > 0 {
		cmd = exec.Command(editor, file, fmt.Sprintf("+%d", lineNum[0]))
	} else {
		cmd = exec.Command(editor, file)
	}
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()
}

func (ai *ArgsInfo) doSearch() int {
	if ai.TaskMainDir == "" {
		logger.Loge("Search disabled (no home directory)")
		return 0
	}
	taskList := module.SearchTasksInModule(ai.TaskMainDir, ai.SearchKeyWords)
	for _, t := range taskList {
		if t[1] != "" {
			logger.Printf("%-32s #%s\n", t[0], t[1])
		} else {
			logger.Printf("%s\n", t[0])
		}
	}
	return 0
}

func (ai *ArgsInfo) doRun() int {
	if ai.TaskMainDir == "" {
		logger.Loge("Module repo disabled (no home directory)")
		return 0
	}
	taskList := module.SearchTasksInModule(ai.TaskMainDir, ai.SearchKeyWords)
	if len(taskList) == 0 {
		logger.Loge("error: no task match!")
		return 1
	}
	return 0
}

func (ai *ArgsInfo) doList() int {
	if ai.File != "" && ai.Task != "" {
		file := ai.expandFile(ai.File)
		info := parser.NewMkInfo(file)
		if info.Err == 0 {
			block := info.FindTaskBlock([]string{ai.Task})
			if block != nil {
				logger.Log(block.CmdBlock)
			} else {
				logger.Logf("Task [%s] DON'T exist!", ai.Task)
				return 1
			}
		} else {
			info.PrintErr()
			return 1
		}
	} else if ai.File != "" {
		if matched, _ := regexp.MatchString(`^@$|^@.+\/$`, ai.File); matched {
			module.PrintRepo(ai.TaskMainDir, ai.File, ai.ListOption)
		} else {
			file := ai.expandFile(ai.File)
			info := parser.NewMkInfo(file)
			if info.Err == 0 {
				info.PrintTask(ai.ListOption)
			} else {
				if ai.ListOption == "" {
					info.PrintErr()
				}
				return 1
			}
		}
	} else {
		return 1
	}
	return 0
}

func (ai *ArgsInfo) doTabComplete() int {
	return 0
}

func (ai *ArgsInfo) doDefault() int {
	if ai.File == "@" || strings.HasSuffix(ai.File, "/") {
		if ai.TaskMainDir == "" {
			logger.Loge("Module repo disabled (no home directory)")
			return 0
		}
		module.PrintRepo(ai.TaskMainDir, ai.File, ai.ListOption)
		return 0
	}

	if ai.File == "" {
		return 1
	}

	file := ai.expandFile(ai.File)
	info := parser.NewMkInfo(file)
	if info.Err > 0 {
		info.PrintErr()
		return 1
	}

	ai.updateRunList()

	if ai.ChangeWorkdir {
		ai.MkfileDir = filepath.Dir(file)
	} else {
		ai.MkfileDir = file
	}

	tasks := []string{}
	if ai.Task != "" {
		tasks = []string{ai.Task}
	} else if len(info.DefaultTasks) > 0 {
		tasks = info.DefaultTasks
	}

	if len(tasks) > 0 {
		block := info.FindTaskBlock(tasks)
		if block == nil {
			logger.Logf("Task [%s] DON'T exist!", ai.Task)
			return 1
		}

		logger.Logi2("Run Task: [" + block.Tasks[0] + "]")

		workdir := ""
		if ai.ChangeWorkdir {
			workdir = filepath.Dir(file)
		}

		runner := executor.NewRunner(ai.TaskMainDir, ai.TmpDir)
		return runner.RunTask(file, block, info.InitBlock, ai.ShellArgs, workdir, filepath.Dir(file))
	}

	info.PrintTask(ai.ListOption)
	return 0
}

func (ai *ArgsInfo) doCompile() int {
	return 0
}

func (ai *ArgsInfo) updateRunList() {
	if ai.TaskMainDir == "" || ai.File != "task.mk" {
		return
	}

	runFileListFname := filepath.Join(ai.TaskMainDir, "run_file_list.txt")
	cwd, _ := os.Getwd()
	mkFullPath := filepath.Join(cwd, ai.File)

	content, err := os.ReadFile(runFileListFname)
	if err != nil {
		os.WriteFile(runFileListFname, []byte(mkFullPath+"\n"), 0o644)
		return
	}

	lines := strings.Split(string(content), "\n")
	found := false
	for _, line := range lines {
		if strings.TrimSpace(line) == mkFullPath {
			found = true
			break
		}
	}

	if !found {
		lines = append(lines, mkFullPath)
		content := strings.Join(lines, "\n")
		os.WriteFile(runFileListFname, []byte(content), 0o644)
	}
}

func (ai *ArgsInfo) GetBaseDir(path string) string {
	if idx := strings.LastIndex(path, "/"); idx > 0 {
		return path[:idx]
	}
	return ""
}
