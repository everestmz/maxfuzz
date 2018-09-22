package supervisor

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/everestmz/maxfuzz/internal/constants"
	"github.com/everestmz/maxfuzz/internal/logging"
	"github.com/everestmz/maxfuzz/internal/storage"

	"github.com/go-cmd/cmd"
	"github.com/mholt/archiver"
)

func commandLogger(command *cmd.Cmd, stop chan bool) {
	for {
		select {
		case line := <-command.Stdout:
			log.Println(line)
		case line := <-command.Stderr:
			log.Println(line)
		case <-stop:
			return
		}
	}
}

func initialFuzzerSetup(target string, l logging.Logger, h storage.StorageHandler) (string, error) {
	targetDir := filepath.Join(constants.LocalTargetDirectory, target)
	syncDir := filepath.Join(constants.LocalSyncDirectory, target)

	// Cleanup old stuff
	os.RemoveAll(targetDir)
	os.MkdirAll(targetDir, 0775)
	os.RemoveAll(syncDir)
	os.MkdirAll(syncDir, 0775)

	// Download and uncompress fuzzer context
	compressedTarget, err := h.GetTarget()
	if err != nil {
		l.Error(fmt.Sprintf("Could not get target: %s", err.Error()))
		return "", err
	}

	err = archiver.Zip.Open(compressedTarget, targetDir)
	if err != nil {
		l.Error(fmt.Sprintf("Could not uncompress target: %s", err.Error()))
		return "", err
	}

	err = os.Remove(compressedTarget)
	if err != nil {
		l.Error(fmt.Sprintf("Could not remove compressed target: %s", err.Error()))
		return "", err
	}

	// Check if any backup exists, and use it instead
	exists, err := h.BackupExists()
	if err != nil {
		l.Error(fmt.Sprintf("Could not check if backup exists: %s", err.Error()))
		return "", err
	}
	if exists {
		err = os.Remove(syncDir)
		os.MkdirAll(syncDir, 0775)
		if err != nil {
			l.Error(fmt.Sprintf("Could not remove compressed target: %s", err.Error()))
			return "", err
		}

		compressedBackup, err := h.GetBackup()
		if err != nil {
			l.Error(fmt.Sprintf("Could not get backup: %s", err.Error()))
			return "", err
		}

		err = archiver.Zip.Open(compressedBackup, syncDir)
		if err != nil {
			l.Error(fmt.Sprintf("Could not uncompress backup: %s", err.Error()))
			return "", err
		}

		err = os.Remove(compressedBackup)
		if err != nil {
			l.Error(fmt.Sprintf("Could not remove compressed backup: %s", err.Error()))
			return "", err
		}

		return "-i- -o /root/fuzz_out", nil
	}
	return "-i /root/fuzz_in -o /root/fuzz_out", nil
}

func setupAFLCmd(env map[string]string, aflIoOptions string) ([]string, error) {
	toReturn := []string{}
	aflBinary, ok := env["AFL_FUZZ"]
	if !ok {
		return toReturn, fmt.Errorf("AFL_FUZZ not populated in environment")
	}
	aflMemoryLimit, ok := env["AFL_MEMORY_LIMIT"]
	if !ok {
		return toReturn, fmt.Errorf("AFL_MEMORY_LIMIT not populated in environment")
	}
	// For now, not used TODO: enable afl extra options
	_, ok = env["AFL_OPTIONS"] //These can be empty
	aflBinaryLocation, ok := env["AFL_BINARY"]
	if !ok {
		return toReturn, fmt.Errorf("AFL_BINARY not populated in environment")
	}
	aflIoOptionsSplit := strings.Split(aflIoOptions, " ")

	if len(aflIoOptionsSplit) == 4 {
		// Starting new fuzzer, we need a sync dir and an in dir
		inDir := aflIoOptionsSplit[1]
		syncDir := aflIoOptionsSplit[3]

		return []string{aflBinary, "-i", inDir, "-o",
			syncDir, "-m", aflMemoryLimit, "--", aflBinaryLocation}, nil
	} else if len(aflIoOptionsSplit) == 3 {
		// Restarting from backup, only need sync dir
		syncDir := aflIoOptionsSplit[2]

		return []string{aflBinary, "-i-", "-o", syncDir,
			"-m", aflMemoryLimit, "--", aflBinaryLocation}, nil
	}

	return nil, fmt.Errorf("Weird AFL_IO_OPTIONS length - is this configured right?")
}

func setupGofuzzCommand(env map[string]string) ([]string, error) {
	toReturn := []string{"/root/go/bin/go-fuzz"}
	goFuzzZip, ok := env["GO_FUZZ_ZIP"]
	if !ok {
		return toReturn, fmt.Errorf("GO_FUZZ_ZIP environment variable not populated")
	}

	toReturn = append(toReturn, fmt.Sprintf("-bin=%s", goFuzzZip), "-workdir=/root/fuzz_out", "-http=0.0.0.0:8000")
	return toReturn, nil
}

// func runCommandWithLogging(c *cmd.Cmd) error {
// 	stop := make(chan bool)
// 	go commandLogger(c, stop)

// 	c.Start()
// 	status := c.Status()
// 	for status.StopTs == 0 {
// 		status = c.Status()
// 	}

// 	stop <- true
// 	if status.Error != nil {
// 		return status.Error
// 	}
// 	if status.Exit != 0 {
// 		return fmt.Errorf("Build steps stopped with exit code %v", status.Exit)
// 	}

// 	return nil
// }
