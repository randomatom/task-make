package module

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"task/internal/logger"
	"task/internal/parser"
)

var reRepoDir = regexp.MustCompile(`^@$|^@.+\/$`)

func SearchTasksInModule(taskMainDir string, keyWords []string) [][2]string {
	taskList := make([][2]string, 0)
	repoDir := filepath.Join(taskMainDir, "repo")

	err := filepath.Walk(repoDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if !info.IsDir() && strings.HasSuffix(path, ".mk") {
			relPath, _ := filepath.Rel(repoDir, path)
			moduleName := strings.TrimSuffix(relPath, ".mk")
			moduleName = strings.ReplaceAll(moduleName, string(filepath.Separator), "/")

			info := parser.NewMkInfo(path)
			tasks := info.SearchTask("@"+moduleName, keyWords)
			taskList = append(taskList, tasks...)
		}
		return nil
	})

	if err != nil {
		return taskList
	}

	return taskList
}

func PrintRepo(taskMainDir, subDir, listOption string) {
	if !reRepoDir.MatchString(subDir) {
		logger.Logf("error: %s", subDir)
		return
	}

	realDir := filepath.Join(taskMainDir, "repo", subDir[1:])

	files, err := os.ReadDir(realDir)
	if err != nil {
		logger.Logf("Module Path [%s] error!", subDir)
		return
	}

	if listOption == "s" {
		for _, f := range files {
			if f.IsDir() {
				logger.Log(subDir + f.Name() + "/")
			} else if strings.HasSuffix(f.Name(), ".mk") {
				name := strings.TrimSuffix(f.Name(), ".mk")
				logger.Log(subDir + name)
			}
		}
	} else {
		for _, f := range files {
			if f.IsDir() {
				logger.Log("    " + subDir + f.Name() + "/")
			} else if strings.HasSuffix(f.Name(), ".mk") {
				name := strings.TrimSuffix(f.Name(), ".mk")
				logger.Log("      " + subDir + name)
			}
		}
	}
}
