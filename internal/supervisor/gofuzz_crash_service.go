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

type GofuzzCrashService struct {
	logger   logging.Logger
	stop     chan bool
	target   string
	revision string
}

func NewGofuzzCrashService(target, revision string, l logging.Logger) GofuzzCrashService {
	return GofuzzCrashService{
		logger:   l,
		stop:     make(chan bool),
		target:   target,
		revision: revision,
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

	storageHandler, err := storage.Init(s.target)
	if err != nil {
		s.logger.Error(fmt.Sprintf("Could not initialize storage client:\n%s", err.Error()))
		return
	}

	s.logger.Info("GofuzzCrashService waiting for crash directories")
	exists := false
	for !exists {
		exists = helpers.Exists(filepath.Join(constants.FuzzerOutputDirectory, "crashers"))
	}

	err = watcher.Watch(filepath.Join(constants.FuzzerOutputDirectory, "crashers"))
	panicOnError(err)

	for {
		select {
		case ev := <-watcher.Event:
			if ev.IsCreate() {
				s.logger.Info("GoFuzzCrashService: Bug found")
				if strings.Contains(ev.Name, ".output") {
					// This is the output of a crash
					// TODO: Parse crash output
				} else {
					// This is a crash payload
					payload := storage.FuzzerPayload{
						Location: ev.Name,
						Category: "CRASH",
						Revision: s.revision,
					}
					crashID := filepath.Base(ev.Name)
					s.logger.Info(fmt.Sprintf("Bug found: %s", crashID))
					_, err = storageHandler.SavePayload(payload)
					if err != nil {
						s.logger.Error(fmt.Sprintf("GofuzzCrashService Could not save bug payload: %s", err.Error()))
					}
				}
			}
		case err := <-watcher.Error:
			s.logger.Error(fmt.Sprintf("GofuzzCrashService: %s", err.Error()))
		case <-s.stop:
			return
		}
	}
}
