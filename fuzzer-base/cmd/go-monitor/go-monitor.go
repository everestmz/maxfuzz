// Manages crash file syncing & uploading for go fuzzers

package main

import (
	"time"

	"maxfuzz/fuzzer-base/internal/helpers"
	"maxfuzz/fuzzer-base/internal/supervisor"

	"github.com/howeyc/fsnotify"
)

var log = helpers.BasicLogger()

func main() {
	// Setup file watchers & uploaders
	crashWatcher, err := fsnotify.NewWatcher()
	helpers.Check("Unable to create crash watcher: %v", err)
	helpers.QuickLog(log, "Created gofuzz crash watcher")

	go helpers.WatchFile(crashWatcher)
	helpers.QuickLog(log, "Started crash watcher goroutine")

	fuzzerSupervisor := supervisor.New("gofuzz-supervisor")
	goFuzzService := &GoService{}
	fuzzerSupervisor.Add(goFuzzService)
	fuzzerSupervisor.ServeBackground()

	// Wait for fuzzers to initialize
	helpers.QuickLog(log, "Waiting for fuzzers to initialize")
	for !helpers.Exists("/root/fuzz_out/crashers") {
		time.Sleep(time.Second * 10)
	}
	helpers.QuickLog(log, "Fuzzer initialized")

	// Add crash and hang directories to file watchers
	err = crashWatcher.Watch("/root/fuzz_out/crashers")
	helpers.Check("Error watching folder: %v", err)
	helpers.QuickLog(log, "Watching gofuzz crash directory")

	// Ensure we backup the fuzz_out dir regularly
	go helpers.RegularBackup("/root/fuzz_out")
	helpers.QuickLog(log, "Started fuzzer state backups goroutine")

	helpers.QuickLog(log, "Fuzzers healthy!")

	//TODO: add fuzzer health checks

	select {}
}
