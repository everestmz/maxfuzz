package supervisor

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/everestmz/maxfuzz/internal/constants"
	"github.com/everestmz/maxfuzz/internal/logging"
	"github.com/everestmz/maxfuzz/internal/storage"

	"github.com/mholt/archiver"
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

	ticker := time.NewTicker(time.Duration(10) * time.Minute)
	for {
		select {
		case <-s.stop:
			ticker.Stop()
			return
		case <-ticker.C:
			outFilePath := storageHandler.GetTargetBackupLocation()
			files, err := filepath.Glob(filepath.Join(constants.LocalSyncDirectory, s.target, "*"))
			if err != nil {
				s.logger.Error(err.Error())
				return
			}
			err = archiver.Zip.Make(outFilePath, files)
			if err != nil {
				s.logger.Error(fmt.Sprintf("Could not compress output for backup:\n%s", err.Error()))
				return
			}
			err = storageHandler.MakeBackup()
			if err != nil {
				s.logger.Error(fmt.Sprintf("Could not make backup:\n%s", err.Error()))
				return
			}
			s.logger.Info("BackupService backup successful")
		}
	}
}
