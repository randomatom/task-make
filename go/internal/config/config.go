package config

import (
	"os"
	"path/filepath"
	"strings"
)

const (
	DefaultTaskDir = ".local/task"
	RepoDirName    = "repo"
)

func homeDir() string {
	home := os.Getenv("HOME")
	if home == "" {
		return ""
	}
	info, err := os.Stat(home)
	if err != nil || !info.IsDir() {
		return ""
	}
	if home == "/" {
		return ""
	}

	normalHome := []string{
		"/home/",
		"/Users/",
		"/root/",
	}
	for _, prefix := range normalHome {
		if strings.HasPrefix(home, prefix) {
			return home
		}
	}

	if info.Mode().Perm()&0o222 == 0 {
		return ""
	}
	return home
}

func GetTaskMainDir() string {
	dir := os.Getenv("_TASK_PROFILE_DIR")
	if dir == "" {
		home := homeDir()
		if home == "" {
			return ""
		}
		dir = filepath.Join(home, DefaultTaskDir)
	}
	return dir
}

func GetRepoDir() string {
	return filepath.Join(GetTaskMainDir(), RepoDirName)
}

func GetTaskRootWorkdir() string {
	workdir := os.Getenv("_TASK_ROOT_WORKDIR")
	if workdir == "" {
		cwd, _ := os.Getwd()
		if len(cwd) > 0 {
			return cwd[:1]
		}
		return ""
	}
	return workdir
}

func GetTmpDir() string {
	tmpDir := os.Getenv("_TASK_TMP_DIR")
	if tmpDir != "" {
		return tmpDir
	}

	user := os.Getenv("USER")
	if user == "" {
		user = "user"
	}

	if _, err := os.Stat("/dev/shm"); err == nil {
		tmpDir = filepath.Join("/dev/shm", "mk_task_dir@"+user)
	} else {
		tmpDir = filepath.Join("/tmp", "mk_task_dir@"+user)
	}

	if _, err := os.Stat(tmpDir); os.IsNotExist(err) {
		if err := os.MkdirAll(tmpDir, 0o700); err != nil {
			return ""
		}
	}

	return tmpDir
}

func GetDefaultTaskFile() string {
	return os.Getenv("_TASK_CUR_DEFAULT_FILE")
}

func IsSubTask() bool {
	return os.Getenv("_TASK_IS_SUBTASK") != ""
}

func EnsureDirs(taskMainDir string) error {
	if taskMainDir == "" {
		return nil
	}
	repoDir := filepath.Join(taskMainDir, RepoDirName)
	for _, dir := range []string{taskMainDir, repoDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return nil
}

func ExpandFile(shortName, repoDir string) string {
	if strings.HasPrefix(shortName, "@") {
		if strings.HasSuffix(shortName, "/") {
			return filepath.Join(repoDir, shortName[1:])
		}
		return filepath.Join(repoDir, shortName[1:]+".mk")
	}
	return shortName
}
