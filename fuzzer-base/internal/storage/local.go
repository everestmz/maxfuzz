package storage

import (
	"fmt"
	"path/filepath"

	"github.com/everestmz/maxfuzz/fuzzer-base/internal/constants"

	"github.com/spf13/afero"
)

type LocalStorageHandler struct {
	targetName string
}

var fs afero.Fs

func initLocalStorage(targetName string) (LocalStorageHandler, error) {
	toReturn := LocalStorageHandler{targetName}
	fs = afero.NewOsFs()
	return toReturn, nil
}

func (h LocalStorageHandler) BackupExists() (bool, error) {
	result, err := afero.Exists(
		fs,
		filepath.Join(
			constants.LocalBackupDirectory, h.targetName, "backup.zip",
		),
	)
	if err != nil {
		return false, err
	}
	return result, nil
}

func (h LocalStorageHandler) filesystemSync(source, destination string) error {
	exists, err := afero.Exists(fs, source)
	if err != nil {
		return fmt.Errorf("File existence check fail: %s", err.Error())
	}
	if exists {
		data, err := afero.ReadFile(fs, source)
		if err != nil {
			return fmt.Errorf("Error reading file: %s", err.Error())
		}
		err = afero.WriteFile(fs, filepath.Join(constants.LocalBackupDirectory, destination), data, 0755)
		if err != nil {
			return fmt.Errorf("Error writing file: %s", err.Error())
		}
		return nil
	}
	return fmt.Errorf("File %s does not exist", source)
}

func (h LocalStorageHandler) filesystemDownload(source, destination string) error {
	exists, err := afero.Exists(fs, source)
	if err != nil {
		return fmt.Errorf("File existence check fail: %s", err.Error())
	}
	if exists {
		data, err := afero.ReadFile(fs, source)
		if err != nil {
			return fmt.Errorf("Error reading file: %s", err.Error())
		}
		err = afero.WriteFile(fs, destination, data, 0755)
		if err != nil {
			return fmt.Errorf("Error writing file: %s", err.Error())
		}
		return nil
	}
	return fmt.Errorf("File %s does not exist", source)
}

func (h LocalStorageHandler) GetBackup() (string, error) {
	backupLocation := filepath.Join(
		constants.LocalBackupDirectory, h.targetName, "backup.zip",
	)

	err := h.filesystemDownload(backupLocation, constants.FuzzerBackupLocation)
	return constants.FuzzerBackupLocation, err
}

func (h LocalStorageHandler) MakeBackup() error {
	destination := filepath.Join(h.targetName, "backup.zip")
	err := h.filesystemSync(constants.FuzzerOutputDirectory, destination)
	return err
}

func (h LocalStorageHandler) SavePayload(source string) error {
	destination := filepath.Join(h.targetName, source)
	err := h.filesystemSync(source, destination)
	return err
}

func (h LocalStorageHandler) SaveStdout(source string) error {
	destination := filepath.Join(h.targetName, source)
	err := h.filesystemSync(source, destination)
	return err
}

func (h LocalStorageHandler) SaveStderr(source string) error {
	destination := filepath.Join(h.targetName, source)
	err := h.filesystemSync(source, destination)
	return err
}

func (h LocalStorageHandler) GetTarget() (string, error) {
	source := filepath.Join(constants.LocalTargetDirectory, fmt.Sprintf("%s.zip", h.targetName))
	destination := filepath.Join(constants.LocalTargetDirectory, "working_target.zip")
	err := h.filesystemDownload(source, destination)
	return destination, err
}
