package main

import (
	"task/internal/args"
	"task/internal/config"
	"task/internal/logger"
	"os"
)

func main() {
	taskMainDir := config.GetTaskMainDir()
	if err := config.EnsureDirs(taskMainDir); err != nil {
		logger.Fatalf("Failed to create task dir: %v", err)
	}

	argInfo := args.NewArgsInfo(os.Args, taskMainDir)
	if argInfo.Err != 0 {
		os.Exit(argInfo.Err)
	}

	ret := argInfo.DoMain()
	os.Exit(ret)
}
