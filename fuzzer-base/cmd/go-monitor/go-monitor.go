// Manages crash file syncing & uploading for go fuzzers

package main

import (
	"net/http"
	"time"

	"maxfuzz/fuzzer-base/internal/helpers"
	"maxfuzz/fuzzer-base/internal/supervisor"

	sse "astuart.co/go-sse"
	"github.com/graphql-go/handler"
	"github.com/howeyc/fsnotify"
)

var log = helpers.BasicLogger()
var startupSteps = 6

func main() {
	// Setup file watchers & uploaders
	crashWatcher, err := fsnotify.NewWatcher()
	helpers.Check("Unable to create crash watcher: %v", err)
	helpers.QuickLog(log, "Created gofuzz crash watcher")
	helpers.LogStartupStep(log, 0, startupSteps, "set up file watchers")

	go helpers.WatchFile(crashWatcher)
	helpers.QuickLog(log, "Started crash watcher goroutine")
	helpers.LogStartupStep(log, 1, startupSteps, "started crash watcher routine")

	fuzzerSupervisor := supervisor.New("gofuzz-supervisor")
	goFuzzService := &GoService{}
	fuzzerSupervisor.Add(goFuzzService)
	fuzzerSupervisor.ServeBackground()
	helpers.LogStartupStep(log, 2, startupSteps, "started fuzzer server")

	// Wait for fuzzers to initialize
	helpers.QuickLog(log, "Waiting for fuzzers to initialize")
	for !helpers.Exists("/root/fuzz_out/crashers") {
		time.Sleep(time.Second * 10)
	}
	helpers.QuickLog(log, "Fuzzer initialized")
	helpers.LogStartupStep(log, 3, startupSteps, "fuzzer output confirmed")

	// Add crash and hang directories to file watchers
	err = crashWatcher.Watch("/root/fuzz_out/crashers")
	helpers.Check("Error watching folder: %v", err)
	helpers.QuickLog(log, "Watching gofuzz crash directory")
	helpers.LogStartupStep(log, 4, startupSteps, "watching crash directories")

	// Ensure we backup the fuzz_out dir regularly
	go helpers.RegularBackup("/root/fuzz_out")
	helpers.QuickLog(log, "Started fuzzer state backups goroutine")
	helpers.LogStartupStep(log, 5, startupSteps, "started fuzzer state backups")

	helpers.QuickLog(log, "Fuzzers healthy!")

	//TODO: add fuzzer health checks

	// Setup & run stats server
	gqlHandler := handler.New(&handler.Config{
		Schema:   &schema,
		Pretty:   true,
		GraphiQL: true,
	})
	http.Handle("/", gqlHandler)

	evCh := make(chan *sse.Event)
	go sse.Notify("http://localhost:8000/eventsource", evCh)
	go func() {
		for {
			updateStats(evCh)
		}
	}()

	helpers.LogStartupStep(log, 6, startupSteps, "stats server setup complete")
	if err = http.ListenAndServe(":8080", nil); err != nil {
		helpers.QuickLog(log, "Server failed to start")
	}
}
