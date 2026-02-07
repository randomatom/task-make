package executor

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"task/internal/logger"
	"task/internal/parser"
	"task/internal/utils"
)

type Runner struct {
	TaskMainDir string
	TmpDir      string
}

func NewRunner(taskMainDir, tmpDir string) *Runner {
	return &Runner{
		TaskMainDir: taskMainDir,
		TmpDir:      tmpDir,
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

	scriptName := fmt.Sprintf("%s/%s_%s%s", r.TmpDir, block.Tasks[0], tag, ext)
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
	tag := utils.GetChecksum(block.CmdBlock)
	scriptName := fmt.Sprintf("%s/%s_%s.sh", r.TmpDir, block.Tasks[0], tag)

	shellCmd := r.buildShellScript(curFile, block, initBlock, newWorkdir, mkfileDir)
	r.writeScript(scriptName, shellCmd)

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

func (r *Runner) buildShellScript(curFile string, block *parser.Block, initBlock *parser.Block, newWorkdir, mkfileDir string) string {
	var sb strings.Builder

	sb.WriteString("set -eu\n")
	sb.WriteString("set -o pipefail\n\n")

	sb.WriteString(fmt.Sprintf("MK_DIR=\"%s\"\n", mkfileDir))

	initFile := r.TaskMainDir + "/__init__.sh"
	if _, err := os.Stat(initFile); err == nil {
		sb.WriteString(fmt.Sprintf(". %s\n", initFile))
	}

	if newWorkdir != "" {
		sb.WriteString(fmt.Sprintf("cd \"%s\"\n", newWorkdir))
	}

	sb.WriteString("\n")

	if initBlock != nil {
		sb.WriteString(strings.TrimSpace(initBlock.CmdBlock))
		sb.WriteString("\n")
	}

	sb.WriteString(strings.TrimSpace(block.CmdBlock))
	sb.WriteString("\n")

	return sb.String()
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
		cmdPath := strings.Join([]string{dir, cmd}, "/")
		if _, err := os.Stat(cmdPath); err == nil {
			return cmdPath
		}
	}
	return ""
}
