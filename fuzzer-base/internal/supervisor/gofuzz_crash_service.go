package supervisor

import (
	"github.com/everestmz/maxfuzz/fuzzer-base/internal/logging"
	"github.com/howeyc/fsnotify"
)

type GofuzzCrashService struct {
	logger logging.Logger
	stop   chan bool
}

func NewGofuzzCrashService(l logging.Logger) GofuzzCrashService {
	return GofuzzCrashService{
		logger: l,
		stop:   make(chan bool),
	}
}

func (s GofuzzCrashService) Stop() {
	s.logger.Info("GofuzzCrashService stopping")
	s.stop <- true
}

func (s GofuzzCrashService) Serve() {
	s.logger.Info("GofuzzCrashService starting")
	watcher, err := fsnotify.NewWatcher()
	panicOnError(err)

	err = watcher.Watch("/root/fuzz_out/crashers")
	panicOnError(err)

	for {
		select {
		case ev := <-watcher.Event:
			if ev.IsCreate() {
				s.logger.Info("GoFuzzCrashService: Bug found")
				// TODO: Handle file event
			}
		case err := <-watcher.Error:
			s.logger.Error(err.Error())
		case <-s.stop:
			return
		}
	}
}
