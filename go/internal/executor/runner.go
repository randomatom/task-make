package executor

import (
	"os"
	"os/exec"
	"strconv"
	"strings"
	"task/internal/logger"
	"task/internal/parser"
	"task/internal/utils"
)

type Runner struct {
	TaskMainDir     string
	TmpDir          string
	ScriptArgs      []string
	IsSubTask       bool
	TaskRootWorkdir string
}

func NewRunner(taskMainDir, tmpDir string) *Runner {
	return &Runner{
		TaskMainDir: taskMainDir,
		TmpDir:      tmpDir,
		IsSubTask:   os.Getenv("_TASK_IS_SUBTASK") != "",
		ScriptArgs:  os.Args,
	}
}

func (r *Runner) RunTask(curFile string, block *parser.Block, initBlock *parser.Block, shellArgs []string, newWorkdir, mkfileDir string) int {
	if block.Interpreter == "bash" || block.Interpreter == "sh" {
		return r.runBashTask(curFile, block, initBlock, shellArgs, newWorkdir, mkfileDir)
	}

	cmdBlock := strings.TrimSpace(block.CmdBlock)
	tag := utils.GetChecksum(cmdBlock)

	interpreterMap := map[string]struct {
		ext string
		cmd string
	}{
		"python":  {".py", "python3"},
		"python3": {".py", "python3"},
		"nu":      {".nu", "nu"},
		"node":    {".js", "node"},
	}

	info := interpreterMap[block.Interpreter]
	ext := info.ext
	cmd := info.cmd

	if !strings.Contains(cmd, "/") {
		cmdPath := r.findInPath(cmd)
		if cmdPath == "" {
			logger.Logf("Interpreter [%s] not found!", cmd)
			return 1
		}
		cmd = cmdPath
	}

	scriptName := r.TmpDir + "/" + block.Tasks[0] + "_" + tag + ext
	r.writeScript(scriptName, cmdBlock)

	execCmd := append([]string{cmd, scriptName}, shellArgs...)
	cmdExec := exec.Command(execCmd[0], execCmd[1:]...)
	cmdExec.Stdin = os.Stdin
	cmdExec.Stdout = os.Stdout
	cmdExec.Stderr = os.Stderr
	cmdExec.Dir = newWorkdir

	if err := cmdExec.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode()
		}
		return 1
	}
	return 0
}

func (r *Runner) runBashTask(curFile string, block *parser.Block, initBlock *parser.Block, shellArgs []string, newWorkdir, mkfileDir string) int {
	block.CmdBlock = strings.TrimSpace(block.CmdBlock)

	esc := "\033[31m"
	res := "\033[0m"

	trapIntFunc := "trap 'onCtrlC ${LINENO}' INT\n" +
		"onCtrlC() {\n" +
		"\t_SIGINT_FLAG=1\n" +
		"\t((line_num=${1}+@2))\n" +
		"\t[ ${1} -lt @1 ] && exit 1\n" +
		"\techo -e \"" + esc + "  SIGINT on [ " + curFile + " +${line_num} ]." + res + "\"\n" +
		"\texit 1\n" +
		"}\n"

	trapErrFunc := "trap 'OnError ${LINENO}' ERR\n" +
		"OnError() {\n" +
		"\terrcode=$?\n" +
		"\t_ERROR_FLAG=1\n" +
		"\tif [ ${1} -ge @2 ] ; then\n" +
		"\t    ((line_num=${1} - @5))\n" +
		"\telif [ ${1} -ge @1 ] ; then\n" +
		"\t    ((line_num=${1} - @4))\n" +
		"\tfi\n" +
		"\t[ ${1} -lt @1 ] && exit 1\n" +
		"\tif [ $errcode -eq 127 ]; then\n" +
		"\t    echo -e \"" + esc + "  Error on [ " + curFile + " +${line_num} ]. " + res + "\"\n" +
		"\t    exit 1\n" +
		"\telse\n" +
		"\t    echo -e \"" + esc + "  Error on [ " + curFile + " +${line_num} ]. code ${errcode}. " + res + "\"\n" +
		"\t    echo -e \"\033[33m  To allow the program to continue running after an error, try \\\"cmd1 | cmd2 || true\\\"\033[0m\"\n" +
		"\t    exit \"${errcode}\"\n" +
		"\tfi\n" +
		"}\n"

	trapExitFunc := "trap 'OnExit ${LINENO}' EXIT\n" +
		"OnExit() {\n" +
		"\terrcode=$?\n" +
		"\t((line_num=${1}+@2))\n" +
		"\t[ ${errcode} -eq 0 ] && exit 0\n" +
		"\t[ -z ${_ERROR_FLAG+x} ] || exit ${errcode}\n" +
		"\t[ -z ${_SIGINT_FLAG+x} ] || exit ${errcode}\n" +
		"\t[ ${1} -lt @1 ] && exit ${errcode}\n" +
		"\t[ ${1} -ge @3 ] && exit 0\n" +
		"\techo -e \"" + esc + "  Exit ${errcode} on [ " + curFile + " +${line_num} ]. " + res + "\"\n" +
		"\texit \"${errcode}\"\n" +
		"}\n"

	trapCmd := trapIntFunc + trapErrFunc + trapExitFunc

	setCmd := "set -eu\n" + "set -o pipefail\n"

	mFuncCmd := ""
	if len(r.ScriptArgs) > 0 {
		if r.ScriptArgs[0] == "m" {
			pathEnv := os.Getenv("PATH")
			if pathEnv != "" {
				for _, dir := range strings.Split(pathEnv, ":") {
					mPath := dir + "/m"
					if _, err := os.Stat(mPath); err == nil {
						mFuncCmd = "m() {\n\t\"" + mPath + "\" \"$@\"\n}\n"
						break
					}
				}
			}
		} else if strings.HasSuffix(r.ScriptArgs[0], ".js") {
			mFuncCmd = "m() {\n\tqjs \"" + r.ScriptArgs[0] + "\" \"$@\"\n}\n"
		} else {
			mFuncCmd = "m() {\n\t\"" + r.ScriptArgs[0] + "\" \"$@\"\n}\n"
		}
	}

	initCmd := "MK_DIR=\"" + mkfileDir + "\"\n"

	if r.TaskMainDir != "" {
		initFile := r.TaskMainDir + "/__init__.sh"
		if _, err := os.Stat(initFile); err == nil {
			initCmd += ". " + initFile + "\n"
		}
	}

	if newWorkdir != "" {
		initCmd += "cd \"" + newWorkdir + "\"\n"
	}

	commentLine := "\n##########################\n\n"

	initBlockCmdBlock := ""
	initBlockStartLineNum := -1
	if initBlock != nil {
		initBlockCmdBlock = strings.TrimRight(initBlock.CmdBlock, " \t\n")
		initBlockStartLineNum = initBlock.StartLineNum
	}

	shellCmd := trapCmd + setCmd + mFuncCmd + initCmd + commentLine

	lines := strings.Split(shellCmd, "\n")
	shTaskInitBlockStart := len(lines)

	initBlockLines := strings.Split(initBlockCmdBlock, "\n")
	initBlockLength := len(initBlockLines)

	userBlockLines := strings.Split(block.CmdBlock, "\n")
	userBlockLength := len(userBlockLines)

	shTaskUserBlockStart := shTaskInitBlockStart + initBlockLength
	shAllLineEnd := shTaskInitBlockStart + initBlockLength + userBlockLength

	shInitBlockOffset := shTaskInitBlockStart - initBlockStartLineNum - 1
	shUserBlockOffset := shTaskUserBlockStart - block.StartLineNum - 1

	shellCmd = strings.ReplaceAll(shellCmd, "@1", strconv.Itoa(shTaskInitBlockStart))
	shellCmd = strings.ReplaceAll(shellCmd, "@2", strconv.Itoa(shTaskUserBlockStart))
	shellCmd = strings.ReplaceAll(shellCmd, "@3", strconv.Itoa(shAllLineEnd))
	shellCmd = strings.ReplaceAll(shellCmd, "@4", strconv.Itoa(shInitBlockOffset))
	shellCmd = strings.ReplaceAll(shellCmd, "@5", strconv.Itoa(shUserBlockOffset))

	if initBlockCmdBlock != "" {
		shellCmd += "\n" + initBlockCmdBlock
	}
	shellCmd += "\n" + block.CmdBlock

	tag := utils.GetChecksum(shellCmd)
	scriptName := r.TmpDir + "/" + block.Tasks[0] + "_" + tag + ".sh"
	r.writeScript(scriptName, shellCmd)

	os.Setenv("_TASK_CUR_DEFAULT_FILE", curFile)
	if r.TaskRootWorkdir != "" {
		os.Setenv("_TASK_ROOT_WORKDIR", r.TaskRootWorkdir)
	}
	os.Setenv("_TASK_TMP_DIR", r.TmpDir)
	os.Setenv("_TASK_IS_SUBTASK", "1")

	execCmd := append([]string{"/bin/bash", scriptName}, shellArgs...)
	cmdExec := exec.Command(execCmd[0], execCmd[1:]...)
	cmdExec.Stdin = os.Stdin
	cmdExec.Stdout = os.Stdout
	cmdExec.Stderr = os.Stderr

	if err := cmdExec.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode()
		}
		return 1
	}
	return 0
}

func (r *Runner) writeScript(path, content string) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		os.WriteFile(path, []byte(content), 0o755)
		return
	}

	existing, _ := os.ReadFile(path)
	if string(existing) != content {
		os.WriteFile(path, []byte(content), 0o755)
	}
}

func (r *Runner) findInPath(cmd string) string {
	pathEnv := os.Getenv("PATH")
	if pathEnv == "" {
		return ""
	}

	for _, dir := range strings.Split(pathEnv, ":") {
		cmdPath := dir + "/" + cmd
		if _, err := os.Stat(cmdPath); err == nil {
			return cmdPath
		}
	}
	return ""
}
