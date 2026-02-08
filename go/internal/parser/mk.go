package parser

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"task/internal/logger"
)

func (mk *MkInfo) ParseFile() {
	file, err := os.Open(mk.File)
	if err != nil {
		mk.Err = 1
		mk.ErrMsg = fmt.Sprintf("%s DON'T exist!", mk.File)
		return
	}
	defer file.Close()

	lines := make([]*LineInfo, 0)
	scanner := bufio.NewScanner(file)
	lineNum := 0
	for scanner.Scan() {
		line := strings.TrimRight(scanner.Text(), " \t")
		lineNum++
		if line == "" {
			continue
		}

		lineInfo := &LineInfo{Num: lineNum, Str: line}
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "###") {
			lineInfo.Type = "div"
		} else if strings.HasPrefix(trimmed, "#!") {
			lineInfo.Type = "pre_cmd"
			if matches := importRegex.FindStringSubmatch(trimmed); len(matches) > 0 {
				lineInfo.Param = []string{"import", matches[1], matches[3]}
			} else if matches := interpreterRegex.FindStringSubmatch(trimmed); len(matches) > 0 {
				lineInfo.Param = []string{"interpreter", matches[1]}
			} else {
				mk.Err = 2
				mk.ErrMsg = fmt.Sprintf("Error at line %d. pre_cmd Error.\n==>    %s", lineNum, line)
				return
			}
		} else if strings.HasPrefix(trimmed, "#") {
			if strings.HasPrefix(trimmed, "###") {
				lineInfo.Type = "div"
			} else {
				continue
			}
		} else if strings.HasSuffix(trimmed, ":") || taskHashBang.MatchString(trimmed) {
			if strings.HasPrefix(trimmed, "__init__:") {
				lineInfo.Type = "task"
				lineInfo.Tasks = []string{"__init__"}
			} else if matches := taskRegex.FindStringSubmatch(trimmed); len(matches) > 0 {
				lineInfo.Type = "task"
				lineInfo.DefaultFlag = matches[1] == "*"
				lineInfo.Tasks = []string{matches[2]}
				if matches[3] != "" {
					lineInfo.Tasks = append(lineInfo.Tasks, matches[3])
				}
				if matches[4] != "" {
					interpreterPath := strings.TrimSpace(matches[4])
					parts := strings.Split(interpreterPath, "/")
					lineInfo.Param = []string{"interpreter", parts[len(parts)-1]}
				}
			} else {
				mk.Err = 2
				mk.ErrMsg = fmt.Sprintf("Error at line %d. The task name is invalid.\n==>    %s", lineNum, line)
				return
			}
		} else if strings.HasPrefix(line, "\t") {
			lineInfo.Type = "code"
			lineInfo.Str = line[1:]
		} else {
			mk.Err = 2
			mk.ErrMsg = fmt.Sprintf("Error at line %d.\n==>    %s", lineNum, line)
			return
		}

		lines = append(lines, lineInfo)
	}

	interpreterMap := make(map[string]string)
	for _, l := range lines {
		if l.Type == "task" && len(l.Param) > 0 && l.Param[0] == "interpreter" {
			if len(l.Tasks) > 0 {
				interpreterMap[l.Tasks[0]] = l.Param[1]
			}
		}
	}

	filteredLines := make([]*LineInfo, 0)
	for _, l := range lines {
		if l.Type != "pre_cmd" {
			filteredLines = append(filteredLines, l)
		}
	}

	rawTaskBlockList := linesIntoGroup(filteredLines)
	if rawTaskBlockList == nil {
		mk.Err = 2
		mk.ErrMsg = "Invalid task block structure"
		return
	}

	taskId := 1
	lastLineNum := 0

	for _, rawTaskBlock := range rawTaskBlockList {
		block := &Block{
			Tasks:       make([]string, 0),
			TaskId:      0,
			CmdBlock:    "",
			Comment:     "",
			Interpreter: "bash",
		}

		for _, line := range rawTaskBlock {
			switch line.Type {
			case "task":
				block.Tasks = line.Tasks
				if line.DefaultFlag {
					if len(mk.DefaultTasks) > 0 {
						mk.Err = 2
						mk.ErrMsg = fmt.Sprintf("Error at line %d: Duplication of default task.\n==>    %s", line.Num, line.Str)
						return
					}
					mk.DefaultTasks = line.Tasks
				}
				if line.Tasks[0] != "__init__" && !strings.HasPrefix(line.Tasks[0], "_") {
					block.TaskId = taskId
					taskId++
				}
				if mk.FindTaskBlock(line.Tasks) != nil {
					mk.Err = 2
					mk.ErrMsg = fmt.Sprintf("Error at line %d. Duplication of task [%v].\n==>    %s", line.Num, line.Tasks, line.Str)
					return
				}
				block.StartLineNum = line.Num
				if len(line.Tasks) > 0 && interpreterMap[line.Tasks[0]] != "" {
					block.Interpreter = interpreterMap[line.Tasks[0]]
				}
			case "code":
				if lastLineNum > 0 {
					if line.Num > lastLineNum+1 {
						for i := lastLineNum + 1; i < line.Num; i++ {
							block.CmdBlock += "\n"
						}
					} else {
						block.CmdBlock += "\n"
					}
				}
				block.CmdBlock += line.Str
				if strings.HasPrefix(strings.TrimSpace(line.Str), "##") {
					block.Comment += strings.TrimPrefix(strings.TrimSpace(line.Str), "##")
				}
			case "div":
				block.WithDiv = true
			}
			lastLineNum = line.Num
		}

		block.CmdBlock = strings.TrimSpace(block.CmdBlock)
		if len(block.Tasks) > 0 && block.Tasks[0] == "__init__" {
			if mk.InitBlock != nil {
				mk.Err = 2
				mk.ErrMsg = "Error: Duplication of task [__int__]."
				return
			}
			mk.InitBlock = block
		} else {
			mk.BlockList = append(mk.BlockList, block)
		}
	}
}

func linesIntoGroup(lines []*LineInfo) [][]*LineInfo {
	if len(lines) == 0 {
		return nil
	}

	blockGroups := make([][]*LineInfo, 0)
	currentBlock := make([]*LineInfo, 0)

	for _, line := range lines {
		if line.Type == "div" {
			if len(currentBlock) > 0 {
				blockGroups = append(blockGroups, currentBlock)
				currentBlock = make([]*LineInfo, 0)
			}
			currentBlock = append(currentBlock, line)
		} else if line.Type == "task" {
			if len(currentBlock) > 0 {
				blockGroups = append(blockGroups, currentBlock)
				currentBlock = make([]*LineInfo, 0)
			}
			currentBlock = append(currentBlock, line)
		} else if line.Type == "code" {
			currentBlock = append(currentBlock, line)
		}
	}

	if len(currentBlock) > 0 {
		blockGroups = append(blockGroups, currentBlock)
	}

	if len(blockGroups) == 0 {
		return nil
	}

	validGroups := make([][]*LineInfo, 0)
	for _, group := range blockGroups {
		hasTask := false
		for _, line := range group {
			if line.Type == "task" {
				hasTask = true
				break
			}
		}
		if hasTask {
			validGroups = append(validGroups, group)
		}
	}

	return validGroups
}

func (mk *MkInfo) SearchTask(moduleName string, keys []string) [][2]string {
	if mk.Err != 0 {
		return nil
	}

	taskList := make([][2]string, 0)
	for _, block := range mk.BlockList {
		taskName := moduleName + ":" + strings.Join(block.Tasks, "/")
		line := taskName
		if block.Comment != "" {
			line += "#" + block.Comment
		}

		find := true
		for _, key := range keys {
			if !strings.Contains(line, key) {
				find = false
				break
			}
		}
		if find {
			taskList = append(taskList, [2]string{taskName, block.Comment})
		}
	}
	return taskList
}

func (mk *MkInfo) FindTaskBlock(tasks []string) *Block {
	for _, t := range tasks {
		if reNumber.MatchString(t) {
			n := 0
			fmt.Sscanf(t, "%d", &n)
			if n == 0 {
				if len(mk.DefaultTasks) > 0 {
					return mk.FindTaskBlock(mk.DefaultTasks)
				}
			} else {
				for _, block := range mk.BlockList {
					if block.TaskId == n {
						return block
					}
				}
			}
		} else {
			for _, block := range mk.BlockList {
				for _, taskName := range block.Tasks {
					if t == taskName {
						return block
					}
				}
			}
		}
	}
	return nil
}

func (mk *MkInfo) PrintErr() {
	if mk.Err > 0 {
		logger.Loge(mk.ErrMsg)
	}
}

func calTaskStrLen(arr []string) int {
	length := 0
	for i, s := range arr {
		length += len(s) + 3
		if i == len(arr)-1 {
			length -= 3
		}
	}
	return length
}

func (mk *MkInfo) PrintTask(listOption string) {
	if listOption == "s" {
		for _, block := range mk.BlockList {
			if len(block.Tasks) > 0 {
				logger.Log(block.Tasks[0])
			}
		}
		return
	}

	logger.Logi("Select a Task:")

	taskMaxLength := 0
	for _, block := range mk.BlockList {
		length := calTaskStrLen(block.Tasks)
		if length > taskMaxLength {
			taskMaxLength = length
		}
	}
	if taskMaxLength > 30 {
		taskMaxLength = 30
	}

	for _, block := range mk.BlockList {
		if listOption == "" && block.WithDiv {
			sepLine := "      ---"
			if len(mk.BlockList) >= 10 {
				sepLine = "      ----"
			}
			for i := 0; i < taskMaxLength; i++ {
				sepLine += "-"
			}
			logger.Log(sepLine)

			startNew := 0
			for _, c := range block.Comment {
				if c == '#' {
					startNew++
				}
			}
			if startNew > 0 && len(block.Comment) >= startNew {
				block.Comment = block.Comment[startNew:]
			}
		}

		if block.TaskId == 0 {
			continue
		}

		line := ""
		if len(mk.DefaultTasks) > 0 && strings.Join(mk.DefaultTasks, "/") == strings.Join(block.Tasks, "/") {
			line += "==>  "
		} else {
			line += "     "
		}

		if block.TaskId <= 9 {
			line += fmt.Sprintf(" %d", block.TaskId)
		} else {
			line += fmt.Sprintf("%d", block.TaskId)
		}
		line += fmt.Sprintf(". %s", block.Tasks[0])
		for i := 1; i < len(block.Tasks); i++ {
			line += fmt.Sprintf(" / %s", block.Tasks[i])
		}

		if block.Comment != "" && listOption == "" {
			curLength := len(line)
			length2 := len(line) - len(fmt.Sprintf(". %s", block.Tasks[0])) + taskMaxLength + 12
			if curLength < length2 {
				for i := 0; i < length2-curLength; i++ {
					line += " "
				}
			}
			line += fmt.Sprintf("# %s", strings.TrimSpace(block.Comment))
		}
		logger.Log(line)
	}
}
