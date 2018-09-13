package supervisor

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/everestmz/maxfuzz/internal/constants"
	"github.com/everestmz/maxfuzz/internal/helpers"
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

func preFuzzCleanup() {
	// TODO: more reliable cleanup. Consider snapshotting the environment & filesystem,
	// and then replacing these in between fuzz cycles. If it takes a few mins, it's
	// definitely worthwhile

	// Remove directories that shouldn't be there but may have been set up by maxfuzz
	os.RemoveAll("/root/TMP_CLANG")
	os.RemoveAll(constants.FuzzerBackupLocation)

	// Clear environment variables
	// Maxfuzz
	os.Unsetenv("BUILD_FILES")
	os.Unsetenv("CORPUS")
	os.Unsetenv("FUZZER_NAME")
	// Sanitizers
	os.Unsetenv("AFL_USE_ASAN")
	os.Unsetenv("ASAN_OPTIONS")
	os.Unsetenv("ASAN_SYMBOLIZER_PATH")
	// AFL
	os.Unsetenv("AFL_FUZZ")
	os.Unsetenv("AFL_BINARY")
	os.Unsetenv("AFL_MEMORY_LIMIT")
	os.Unsetenv("AFL_OPTIONS")
	// C/C++
	os.Unsetenv("LD_PRELOAD")
}

func initialFuzzerSetup(l logging.Logger, h storage.StorageHandler) error {
	// Download and uncompress fuzzer context
	compressedTarget, err := h.GetTarget()
	if err != nil {
		l.Error(fmt.Sprintf("Could not get target: %s", err.Error()))
		return err
	}

	err = archiver.Zip.Open(compressedTarget, constants.FuzzerLocation)
	if err != nil {
		l.Error(fmt.Sprintf("Could not uncompress target: %s", err.Error()))
		return err
	}

	err = os.Remove(compressedTarget)
	if err != nil {
		l.Error(fmt.Sprintf("Could not remove compressed target: %s", err.Error()))
		return err
	}

	// Check if any backup exists, and use it instead
	exists, err := h.BackupExists()
	if err != nil {
		l.Error(fmt.Sprintf("Could not check if backup exists: %s", err.Error()))
		return err
	}
	if exists {
		err = os.Remove(constants.FuzzerOutputDirectory)
		if err != nil {
			l.Error(fmt.Sprintf("Could not remove compressed target: %s", err.Error()))
			return err
		}

		compressedBackup, err := h.GetBackup()
		if err != nil {
			l.Error(fmt.Sprintf("Could not get backup: %s", err.Error()))
			return err
		}

		err = archiver.Zip.Open(compressedBackup, constants.FuzzerOutputDirectory)
		if err != nil {
			l.Error(fmt.Sprintf("Could not uncompress backup: %s", err.Error()))
			return err
		}

		err = os.Remove(compressedBackup)
		if err != nil {
			l.Error(fmt.Sprintf("Could not remove compressed backup: %s", err.Error()))
			return err
		}

		err = os.Remove(constants.AFLIOOptions)
		if err != nil {
			l.Error(fmt.Sprintf("Could not remove afl-io-options: %s", err.Error()))
			return err
		}

		f, err := os.Create(constants.AFLIOOptions)
		if err != nil {
			l.Error(fmt.Sprintf("Could not create afl-io-options: %s", err.Error()))
			return err
		}

		toWrite := []byte("-i- -o /root/fuzz_out")
		os.Setenv("AFL_IO_OPTIONS", "-i- -o /root/fuzz_out")
		_, err = f.Write(toWrite)
		if err != nil {
			l.Error(fmt.Sprintf("Could not write to afl-io-options: %s", err.Error()))
			return err
		}
		f.Close()
	} else {
		os.Setenv("AFL_IO_OPTIONS", "-i /root/fuzz_in -o /root/fuzz_out")
	}

	return err
}

func setupAFLCmd() *cmd.Cmd {
	var command *cmd.Cmd
	aflBinary := helpers.GetenvOrDie("AFL_FUZZ")
	aflIoOptions := helpers.GetenvOrDie("AFL_IO_OPTIONS")
	aflMemoryLimit := helpers.GetenvOrDie("AFL_MEMORY_LIMIT")
	// For now, not used TODO: enable afl extra options
	_ = helpers.Getenv("AFL_OPTIONS", "") //These can be empty
	aflBinaryLocation := helpers.GetenvOrDie("AFL_BINARY")
	aflIoOptionsSplit := strings.Split(aflIoOptions, " ")

	if len(aflIoOptionsSplit) == 4 {
		// Starting new fuzzer, we need a sync dir and an in dir
		inDir := aflIoOptionsSplit[1]
		syncDir := aflIoOptionsSplit[3]

		command = cmd.NewCmdOptions(
			aflCmdOptions, aflBinary, "-i", inDir, "-o",
			syncDir, "-m", aflMemoryLimit, "--", aflBinaryLocation)
	} else if len(aflIoOptionsSplit) == 3 {
		// Restarting from backup, only need sync dir
		syncDir := aflIoOptionsSplit[2]

		command = cmd.NewCmdOptions(
			aflCmdOptions, aflBinary, "-i-", "-o", syncDir,
			"-m", aflMemoryLimit, "--", aflBinaryLocation)
	}

	return command
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
