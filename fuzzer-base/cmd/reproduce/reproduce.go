package main

import (
	"os"
	"time"

	"maxfuzz/fuzzer-base/internal/helpers"

	"github.com/adjust/rmq"
	"github.com/howeyc/fsnotify"
)

var log = helpers.BasicLogger()

func main() {
	helpers.QuickLog(log, "Starting crash reproducer")
	slaveWatcher, err := fsnotify.NewWatcher()
	helpers.Check("Unable to create slave watcher: %v", err)
	masterWatcher, err := fsnotify.NewWatcher()
	helpers.Check("Unable to create master watcher: %v", err)
	helpers.QuickLog(log, "Created crash reproduction watchers")

	go helpers.WatchFile(masterWatcher)
	go helpers.WatchFile(slaveWatcher)
	helpers.QuickLog(log, "Started watching for output")

	os.MkdirAll("/root/fuzz_out/master/crashes", 0755)
	os.MkdirAll("/root/fuzz_out/slave/crashes", 0755)

	// Add crash and hang directories to file watchers
	err = masterWatcher.Watch("/root/fuzz_out/master/crashes")
	helpers.Check("Error watching folder: %v", err)

	err = slaveWatcher.Watch("/root/fuzz_out/slave/crashes")
	helpers.Check("Error watching folder: %v", err)

	connection := rmq.OpenConnection(
		"crash stream", "tcp",
		helpers.Getenv("REDIS_QUEUE_URL", ""), 1)

	crashQueue := connection.OpenQueue(helpers.GetFuzzerName())
	helpers.QuickLog(log, "Opened crash queue connection")

	crashConsumer := &(CrashConsumer{})
	crashQueue.StartConsuming(1, time.Second)
	crashQueue.AddConsumer("crash consumer", crashConsumer)
	helpers.QuickLog(log, "Starting to reproduce...")

	select {}
}
