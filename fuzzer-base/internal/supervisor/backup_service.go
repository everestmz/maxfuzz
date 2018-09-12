package supervisor

import (
	"github.com/everestmz/maxfuzz/fuzzer-base/internal/logging"
)

type BackupService struct {
	logger logging.Logger
	stop   chan bool
}

func NewBackupService(l logging.Logger) BackupService {
	return BackupService{
		logger: l,
		stop:   make(chan bool),
	}
}

func (s BackupService) Stop() {
	s.logger.Info("BackupService stopping")
	s.stop <- true
}

func (s BackupService) Serve() {
	s.logger.Info("BackupService starting")
	for {
		select {
		case <-s.stop:
			return
		}
		// helpers.RegularBackup(constants.FuzzerOutputDirectory)
	}
}
