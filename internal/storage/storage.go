package storage

import (
	"fmt"

	"github.com/everestmz/maxfuzz/internal/helpers"
	"github.com/everestmz/maxfuzz/internal/logging"
)

type StorageHandler interface {
	GetTarget() (string, error)
	BackupExists() (bool, error)
	GetBackup() (string, error)
	MakeBackup() error
	// TODO: For save payload/output append timestamp so we dont overwrite on filesystem
	SavePayload(string) error
	SaveOutput(string) error
	GetTargetBackupLocation() string
}

var soln StorageHandler

// Init sets up connections to whatever storage mechanism is being used
func Init(targetName string) (StorageHandler, error) {
	var err error
	opts := helpers.MaxfuzzOptions()
	val, ok := opts["storageSolution"]
	if !ok {
		return nil, fmt.Errorf("No storageSolution specified in MAXFUZZ_OPTIONS")
	}

	switch val {
	case "local":
		soln, err = initLocalStorage(targetName)
		if err != nil {
			return nil, err
		}

		return soln, nil
	default:
		return nil, fmt.Errorf("Invalid storageSolution in MAXFUZZ_OPTIONS")
	}
}

func check(msg string, err error, l logging.Logger) {
	if err != nil {
		l.Error(fmt.Sprintf("Storage error: %s", msg))
		panic(err)
	}
}
