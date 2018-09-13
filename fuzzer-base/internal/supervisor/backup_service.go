package supervisor

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/mholt/archiver"

	"github.com/everestmz/maxfuzz/fuzzer-base/internal/constants"
	"github.com/everestmz/maxfuzz/fuzzer-base/internal/logging"
	"github.com/everestmz/maxfuzz/fuzzer-base/internal/storage"
)

type BackupService struct {
	logger logging.Logger
	stop   chan bool
	target string
}

func NewBackupService(target string, l logging.Logger) BackupService {
	return BackupService{
		logger: l,
		stop:   make(chan bool),
		target: target,
	}
}

func (s BackupService) Stop() {
	s.logger.Info("BackupService stopping")
	s.stop <- true
}

func (s BackupService) Serve() {
	s.logger.Info("BackupService starting")
	storageHandler, err := storage.Init(s.target)
	if err != nil {
		s.logger.Error(fmt.Sprintf("Could not initialize storage client:\n%s", err.Error()))
		return
	}
	for {
		timer := time.NewTimer(time.Duration(10) * time.Minute)
		select {
		case <-s.stop:
			return
		case <-timer.C:
			outFilePath := constants.FuzzerBackupLocation
			files, err := filepath.Glob(filepath.Join(constants.FuzzerOutputDirectory, "*"))
			if err != nil {
				s.logger.Error(err.Error())
				return
			}
			err = archiver.Zip.Make(outFilePath, files)
			if err != nil {
				s.logger.Error(fmt.Sprintf("Could not compress output for backup:\n%s", err.Error()))
				return
			}
			storageHandler.MakeBackup()
		}

	}
}
