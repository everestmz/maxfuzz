package supervisor

import (
	"github.com/everestmz/maxfuzz/fuzzer-base/internal/logging"
	"github.com/howeyc/fsnotify"
)

type AFLCrashService struct {
	logger logging.Logger
	stop   chan bool
}

func NewAFLCrashService(l logging.Logger) AFLCrashService {
	return AFLCrashService{
		logger: l,
		stop:   make(chan bool),
	}
}

func (s AFLCrashService) Stop() {
	s.logger.Info("AFLCrashService stopping")
	s.stop <- true
}

func (s AFLCrashService) Serve() {
	s.logger.Info("AFLCrashService starting")
	_, err := fsnotify.NewWatcher()
	panicOnError(err)

	// err = watcher.Watch("/root/fuzz_out/crashes")
	// panicOnError(err)

	// err = watcher.Watch("/root/fuzz_out/hangs")
	// panicOnError(err)

	for {
		select {
		case <-s.stop:
			return
		}
		// select {
		// case ev := <-watcher.Event:
		// 	if ev.IsCreate() {
		// 		s.logger.Info("Bug found")
		// 		// TODO: Handle file event
		// 	}
		// case err := <-watcher.Error:
		// 	s.logger.Error(err.Error())
		// }
	}
}
