package supervisor

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/everestmz/maxfuzz/internal/constants"
	"github.com/everestmz/maxfuzz/internal/docker"
	"github.com/everestmz/maxfuzz/internal/helpers"
	"github.com/everestmz/maxfuzz/internal/logging"
	"github.com/everestmz/maxfuzz/internal/storage"
	"github.com/everestmz/maxfuzz/internal/types"

	"github.com/go-cmd/cmd"
	"github.com/subosito/gotenv"
	"github.com/thejerf/suture"
)

type CFuzzerService struct {
	logger     logging.Logger
	targetID   string
	targetName string
	stop       chan bool
	baseImage  string
}

var aflCmdOptions = cmd.Options{
	Buffered:  false,
	Streaming: true,
}

func NewCFuzzer(target *types.Target, stats chan *TargetStats) *suture.Supervisor {
	log := logging.NewTargetLogger(target.Name)
	ret := New(log, target.Name)
	ret.Add(NewBackupService(target.UniqueID, log))
	ret.Add(NewAFLStatsService(target.UniqueID, log, stats))
	ret.Add(NewAFLCrashService(target.UniqueID, target.Revision, log))
	ret.Add(CFuzzerService{
		log,
		target.UniqueID,
		target.Name,
		make(chan bool),
		"fuzzbox_c",
	})
	return ret
}

func (s CFuzzerService) Stop() {
	s.logger.Info(fmt.Sprintf("CFuzzerService stopping"))
	s.stop <- true
	s.logger.Info(fmt.Sprintf("CFuzzerService stopped"))
}

func (s CFuzzerService) Serve() {
	s.logger.Info(fmt.Sprintf("CFuzzerService starting"))
	storageHandler, err := storage.Init(s.targetID)
	if err != nil {
		s.logger.Error(fmt.Sprintf("CFuzzerService could not initialize storageHandler: %s", err.Error()))
		return
	}

	// Pre-run sync and download steps
	s.logger.Info(fmt.Sprintf("CFuzzerService setting up target"))
	aflIoOptions, err := initialFuzzerSetup(s.targetID, s.logger, storageHandler)
	if err != nil {
		s.logger.Error(fmt.Sprintf("CFuzzerService could not initialize fuzzer: %s", err.Error()))
		return
	}

	// Get environment
	environmentFile, err := os.Open(filepath.Join(constants.LocalTargetDirectory, s.targetID, "environment"))
	if err != nil {
		s.logger.Error(fmt.Sprintf("CFuzzerService could not parse the environment: %s", err.Error()))
		return
	}
	environment := gotenv.Parse(environmentFile)

	// Run the build steps
	opts := helpers.MaxfuzzOptions()
	suppress := opts["suppressFuzzerOutput"] == "1"
	stdout := stdoutWriter{
		suppressOutput: suppress,
		target:         s.targetName,
	}
	stderr := stderrWriter{
		suppressOutput: suppress,
		target:         s.targetName,
	}
	s.logger.Info(fmt.Sprintf("CFuzzerService running build steps"))
	config, err := docker.CreateFuzzer(s.targetID, s.baseImage, s.stop, map[string]string{}, stdout, stderr)
	if err != nil {
		s.logger.Error(fmt.Sprintf("CFuzzerService could not build the fuzzer: %s", err.Error()))
		return
	}

	// Finally, run the fuzzer
	s.logger.Info(fmt.Sprintf("CFuzzerService running fuzzer"))
	command, err := setupAFLCmd(environment, aflIoOptions)
	if err != nil {
		s.logger.Error(fmt.Sprintf("CFuzzerService could not set up the fuzz command: %s", err.Error()))
		return
	}

	fuzzCluster, err := config.Deploy(command, stdout, stderr)
	if err != nil {
		s.logger.Error(fmt.Sprintf("CFuzzerService could not start the fuzzer: %s", err.Error()))
		return
	}

	clusterState, err := fuzzCluster.State()
	if err != nil {
		s.logger.Error(fmt.Sprintf("CFuzzerService could not start the fuzzer: %s", err.Error()))
		return
	}

	ticker := time.NewTicker(time.Second)
	for {
		select {
		case <-s.stop:
			s.logger.Info(fmt.Sprintf("CFuzzerService spinning down fuzzer"))
			ticker.Stop()
			err = fuzzCluster.Kill()
			if err != nil {
				s.logger.Error(fmt.Sprintf("CFuzzerService could not spin down the fuzzer: %s", err.Error()))
			}
			return
		case <-ticker.C:
			clusterState, err = fuzzCluster.State()
			if err != nil {
				s.logger.Error(fmt.Sprintf("CFuzzerService could not start the fuzzer: %s", err.Error()))
				return
			}
			if !clusterState.Running() {
				s.logger.Error(
					fmt.Sprintf(
						"CFuzzerService fuzz cluster stopped unexpectedly\nErrors: %s\nExit code: %v",
						err.Error(), clusterState.ExitCode()))
				return
			}
		}
	}
}
