// Manages crash file uploading & syncing, and logging, for all AFL fuzzers

package main

import (
	"net/http"
	"time"

	"maxfuzz/fuzzer-base/internal/helpers"
	"maxfuzz/fuzzer-base/internal/supervisor"

	"github.com/graphql-go/handler"
	"github.com/howeyc/fsnotify"
)

var log = helpers.BasicLogger()
var startupSteps = 6

func main() {
	// Setup file watchers & uploaders
	masterWatcher, err := fsnotify.NewWatcher()
	helpers.Check("Unable to create master watcher: %v", err)
	helpers.QuickLog(log, "Created master crash watcher")
	slaveWatcher, err := fsnotify.NewWatcher()
	helpers.Check("Unable to create slave watcher: %v", err)
	helpers.QuickLog(log, "Created slave crash watcher")
	helpers.LogStartupStep(log, 0, startupSteps, "set up file watchers")

	go helpers.WatchFile(masterWatcher)
	go helpers.WatchFile(slaveWatcher)
	helpers.QuickLog(log, "Started crash watcher goroutines")
	helpers.LogStartupStep(log, 1, startupSteps, "started crash watcher routines")

	// Start up supervisor and begin running fuzzers
	fuzzerSupervisor := supervisor.New("fuzzer-ctl")
	masterService := &AFLService{"master"}
	slaveService := &AFLService{"slave"}
	fuzzerSupervisor.Add(masterService)
	fuzzerSupervisor.Add(slaveService)
	fuzzerSupervisor.ServeBackground()
	helpers.LogStartupStep(log, 2, startupSteps, "started fuzzer server")

	// Wait for fuzzers to initialize
	helpers.QuickLog(log, "Waiting for fuzzers to initialize")
	for !helpers.Exists("/root/fuzz_out/master/crashes") ||
		!helpers.Exists("/root/fuzz_out/slave/crashes") {
		time.Sleep(time.Second * 1)
	}
	helpers.QuickLog(log, "Fuzzers initialized")
	helpers.LogStartupStep(log, 3, startupSteps, "fuzzer output confirmed")

	// Add crash and hang directories to file watchers
	err = masterWatcher.Watch("/root/fuzz_out/master/crashes")
	helpers.Check("Error watching folder: %v", err)
	err = masterWatcher.Watch("/root/fuzz_out/master/hangs")
	helpers.Check("Error watching folder: %v", err)
	helpers.QuickLog(log, "Watching master crash directories")

	err = slaveWatcher.Watch("/root/fuzz_out/slave/crashes")
	helpers.Check("Error watching folder: %v", err)
	err = slaveWatcher.Watch("/root/fuzz_out/slave/hangs")
	helpers.Check("Error watching folder: %v", err)
	helpers.QuickLog(log, "Watching slave crash directories")
	helpers.LogStartupStep(log, 4, startupSteps, "watching crash directories")

	// Ensure we backup the fuzz_out dir regularly
	go helpers.RegularBackup("/root/fuzz_out")
	helpers.QuickLog(log, "Started fuzzer state backups goroutine")
	helpers.LogStartupStep(log, 5, startupSteps, "started fuzzer state backups")

	// Ensure the fuzzers are outputting stats
	helpers.QuickLog(log, "Waiting for fuzzer health check to complete")
	for !helpers.Exists("/root/fuzz_out/master/fuzzer_stats") ||
		!helpers.Exists("/root/fuzz_out/slave/fuzzer_stats") {
		time.Sleep(time.Second * 1)
	}
	helpers.QuickLog(log, "Fuzzers healthy!")

	helpers.QuickLog(log, "Starting fuzzer stats server")
	gqlHandler := handler.New(&handler.Config{
		Schema:   &schema,
		Pretty:   true,
		GraphiQL: true,
	})
	http.Handle("/", gqlHandler)

	go func() {
		for {
			time.Sleep(time.Second * 5)
			helpers.QuickLog(log, "Updating stats...")
			updateStats()
			helpers.QuickLog(log, "Stats successfully updated")
		}
	}()

	helpers.LogStartupStep(log, 6, startupSteps, "stats server setup complete")
	if err = http.ListenAndServe(":8080", nil); err != nil {
		helpers.QuickLog(log, "Server failed to start")
	}
}
