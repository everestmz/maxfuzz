package supervisor

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/everestmz/maxfuzz/internal/constants"
	"github.com/everestmz/maxfuzz/internal/helpers"
	"github.com/everestmz/maxfuzz/internal/logging"
	"github.com/everestmz/maxfuzz/internal/storage"

	"github.com/howeyc/fsnotify"
)

type AFLCrashService struct {
	logger logging.Logger
	stop   chan bool
	target string
}

func NewAFLCrashService(target string, l logging.Logger) AFLCrashService {
	return AFLCrashService{
		logger: l,
		stop:   make(chan bool),
		target: target,
	}
}

func (s AFLCrashService) Stop() {
	s.logger.Info("AFLCrashService stopping")
	s.stop <- true
}

func (s AFLCrashService) Serve() {
	s.logger.Info("AFLCrashService starting")
	storageHandler, err := storage.Init(s.target)
	if err != nil {
		s.logger.Error(fmt.Sprintf("Could not initialize storage client:\n%s", err.Error()))
		return
	}

	watchDirectories := []string{
		filepath.Join(constants.LocalSyncDirectory, s.target, "crashes"),
		filepath.Join(constants.LocalSyncDirectory, s.target, "hangs"),
	}

	watcher, err := fsnotify.NewWatcher()
	panicOnError(err)

	// Wait for AFL crash directories to exist
	s.logger.Info("AFLCrashService waiting for crash directories")
	exists := false
	for !exists {
		exists = true
		for _, d := range watchDirectories {
			if !helpers.Exists(d) {
				exists = false
			}
		}
	}

	for _, d := range watchDirectories {
		err = watcher.Watch(d)
		panicOnError(err)
	}

	s.logger.Info("AFLCrashService watching crash directories")
	for {
		select {
		case ev := <-watcher.Event:
			if ev.IsCreate() && !strings.Contains(ev.Name, "README.txt") {
				crashID := filepath.Base(ev.Name)
				s.logger.Info(fmt.Sprintf("Bug found: %s", crashID))
				err = storageHandler.SavePayload(ev.Name)
				if err != nil {
					s.logger.Error(fmt.Sprintf("AFLCrashService Could not save bug payload: %s", err.Error()))
				}
			}
		case err := <-watcher.Error:
			s.logger.Error(fmt.Sprintf("AFLCrashService: %s", err.Error()))
		case <-s.stop:
			return
		}
	}
}
