package main

import (
  "fmt"
  "os"

  "maxfuzz/fuzzer-base/internal/helpers"

  "github.com/go-cmd/cmd"
)

var cmdOptions = cmd.Options{
  Buffered: false,
  Streaming: true,
}

type GoService struct {}

func (s GoService) Stop() {
  helpers.QuickLog(log, "Stopping gofuzz instance")
}

func (s GoService) Serve() {
  helpers.QuickLog(log, "Starting gofuzz instance")
  goFuzzBinary := "/root/go/bin/go-fuzz"
  goFuzzZip := fmt.Sprintf("-bin=%s", helpers.GetenvOrDie("GO_FUZZ_ZIP"))
  goFuzzWorkdir := "-workdir=/root/fuzz_out"

  goFuzzRunCommand := fmt.Sprintf("%s %s %s", goFuzzBinary,
    goFuzzZip, goFuzzWorkdir)

  command := cmd.NewCmdOptions(cmdOptions, goFuzzBinary, goFuzzZip,
    goFuzzWorkdir)

  helpers.QuickLog(log, fmt.Sprintf("Running gofuzz command: %s",
    goFuzzRunCommand))

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

  helpers.QuickLog(log, "Started log interception")
  command.Start()
  helpers.QuickLog(log, "Started fuzzer")
  status := command.Status()
  for status.StopTs == 0 {
		status = command.Status()
	}
  helpers.QuickLog(log, fmt.Sprintf("gofuzz died with exit code: %v \n %s",
    status.Exit, status.Error))
}
