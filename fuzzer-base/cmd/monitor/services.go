package main

import (
	"fmt"
	"os"
	"strings"

	"maxfuzz/fuzzer-base/internal/helpers"

	"github.com/go-cmd/cmd"
)

var cmdOptions = cmd.Options{
	Buffered:  false,
	Streaming: true,
}

type AFLService struct {
	Instance string //master or slave
}

func (s AFLService) Stop() {
	helpers.QuickLog(log, fmt.Sprintf("Stopping AFL %s instance", s.Instance))
}

func (s AFLService) Serve() {
	helpers.QuickLog(log, fmt.Sprintf("Starting AFL %s instance", s.Instance))
	aflBinary := helpers.GetenvOrDie("AFL_FUZZ")
	aflIoOptions := helpers.GetenvOrDie("AFL_IO_OPTIONS")
	aflMemoryLimit := helpers.GetenvOrDie("AFL_MEMORY_LIMIT")
	aflExtraOptions := helpers.Getenv("AFL_OPTIONS", "") //These can be empty
	aflBinaryLocation := helpers.GetenvOrDie("AFL_BINARY")
	aflInstanceFlag := strings.ToUpper(string(s.Instance[0]))

	aflIoOptionsSplit := strings.Split(aflIoOptions, " ")
	aflRunCommand := ""
	var command *cmd.Cmd

	if (len(aflIoOptionsSplit) == 4) {
		// Starting new fuzzer, we need a sync dir and an in dir
		helpers.QuickLog(log, "Starting new fuzzer from scratch")
		inDir := aflIoOptionsSplit[1]
		syncDir := aflIoOptionsSplit[3]

		aflRunCommand = fmt.Sprintf(
			"%s %s -m %s %s -%s %s -- %s",
			aflBinary, aflIoOptions, aflMemoryLimit,
			aflExtraOptions, aflInstanceFlag, s.Instance,
			aflBinaryLocation,
		)

		command = cmd.NewCmdOptions(cmdOptions,
			aflBinary, "-i", inDir, "-o", syncDir, "-m", aflMemoryLimit,
			fmt.Sprintf("-%s", aflInstanceFlag), s.Instance, "--", aflBinaryLocation)
	} else if (len(aflIoOptionsSplit) == 3) {
		// Restarting from backup, only need sync dir
		helpers.QuickLog(log, "Restarting fuzzer from backup")
		syncDir := aflIoOptionsSplit[2]

		aflRunCommand = fmt.Sprintf(
			"%s %s -m %s %s -%s %s -- %s",
			aflBinary, aflIoOptions, aflMemoryLimit,
			aflExtraOptions, aflInstanceFlag, s.Instance,
			aflBinaryLocation,
		)

		command = cmd.NewCmdOptions(cmdOptions,
			aflBinary, "-i-", "-o", syncDir, "-m", aflMemoryLimit,
			fmt.Sprintf("-%s", aflInstanceFlag), s.Instance, "--", aflBinaryLocation)
	}

	helpers.QuickLog(log, fmt.Sprintf("Running afl command: %s", aflRunCommand))

	// Kick off logger
	go func() {
		for {
			select {
			case line := <-command.Stdout:
				fmt.Println(line)
			case line := <-command.Stderr:
				fmt.Fprintln(os.Stderr, line)
			}
		}
	}()
	helpers.QuickLog(log, fmt.Sprintf("%s: Started log interception", s.Instance))
	command.Start()
	helpers.QuickLog(log, fmt.Sprintf("%s: Started fuzzer", s.Instance))
	status := command.Status()
	for status.StopTs == 0 {
		status = command.Status()
	}
	helpers.QuickLog(log, fmt.Sprintf("%s: died with exit code: %v \n %s", s.Instance, status.Exit, status.Error))
}
