package supervisor

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/everestmz/maxfuzz/internal/constants"
	"github.com/everestmz/maxfuzz/internal/docker"
	"github.com/everestmz/maxfuzz/internal/helpers"
	"github.com/everestmz/maxfuzz/internal/logging"
	"github.com/everestmz/maxfuzz/internal/storage"
	"github.com/subosito/gotenv"

	"github.com/thejerf/suture"
)

type GoFuzzerService struct {
	logger    logging.Logger
	targetID  string
	stop      chan bool
	baseImage string
	statsPort string
}

var availableHostPorts map[int]bool

func getAvailablePort() string {
	if availableHostPorts == nil {
		availableHostPorts = map[int]bool{}
	}
	base := 8090
	for {
		available, ok := availableHostPorts[base]
		if !ok || available {
			availableHostPorts[base] = false
			return fmt.Sprintf("%v", base)
		}
		base += 1
	}
}

func unregisterPort(port string) error {
	if availableHostPorts == nil {
		availableHostPorts = map[int]bool{}
	}
	var portNum int
	var err error
	portNum, err = strconv.Atoi(port)
	if err != nil {
		return err
	}
	availableHostPorts[portNum] = true
	return nil
}

func NewGoFuzzer(target string, stats chan *TargetStats) *suture.Supervisor {
	log := logging.NewTargetLogger(target)
	ret := New(log, target)
	statsPort := getAvailablePort()
	ret.Add(NewBackupService(target, log))
	ret.Add(NewGofuzzStatsService(target, statsPort, log, stats))
	ret.Add(NewGofuzzCrashService(target, log))
	ret.Add(GoFuzzerService{
		log, target, make(chan bool), "fuzzbox_go", statsPort,
	})
	return ret
}

func (s GoFuzzerService) Stop() {
	s.logger.Info("CFuzzerService stopping")
	s.stop <- true
	s.logger.Info("CFuzzerService stopped")
}

func (s GoFuzzerService) Serve() {
	s.logger.Info(fmt.Sprintf("GoFuzzerService starting"))
	storageHandler, err := storage.Init(s.targetID)
	if err != nil {
		s.logger.Error(fmt.Sprintf("GoFuzzerService could not initialize storageHandler: %s", err.Error()))
		return
	}

	// Pre-run sync and download steps
	s.logger.Info(fmt.Sprintf("GoFuzzerService setting up target"))
	_, err = initialFuzzerSetup(s.targetID, s.logger, storageHandler)
	if err != nil {
		s.logger.Error(fmt.Sprintf("GouzzerService could not initialize fuzzer: %s", err.Error()))
		return
	}

	// Get environment
	environmentFile, err := os.Open(filepath.Join(constants.LocalTargetDirectory, s.targetID, "environment"))
	if err != nil {
		s.logger.Error(fmt.Sprintf("GoFuzzerService could not parse the environment: %s", err.Error()))
		return
	}
	environment := gotenv.Parse(environmentFile)

	// Run the build steps
	opts := helpers.MaxfuzzOptions()
	suppress := opts["suppressFuzzerOutput"] == "1"
	stdout := stdoutWriter{
		suppressOutput: suppress,
		target:         s.targetID,
	}
	stderr := stderrWriter{
		suppressOutput: suppress,
		target:         s.targetID,
	}
	s.logger.Info(fmt.Sprintf("GoFuzzerService running build steps"))
	config, err := docker.CreateFuzzer(s.targetID, s.baseImage, s.stop, map[string]string{
		"8000": s.statsPort, // Expose the gofuzz stats port
	}, stdout, stderr)
	if err != nil {
		s.logger.Error(fmt.Sprintf("GoFuzzerService could not build the fuzzer: %s", err.Error()))
		return
	}

	// Finally, run the fuzzer
	s.logger.Info(fmt.Sprintf("GoFuzzerService running fuzzer"))
	command, err := setupGofuzzCommand(environment)
	if err != nil {
		s.logger.Error(fmt.Sprintf("GoFuzzerService could not set up the fuzz command: %s", err.Error()))
		return
	}

	fuzzCluster, err := config.Deploy(command, stdout, stderr)
	if err != nil {
		s.logger.Error(fmt.Sprintf("GoFuzzerService could not start the fuzzer: %s", err.Error()))
		return
	}

	clusterState, err := fuzzCluster.State()
	if err != nil {
		s.logger.Error(fmt.Sprintf("GoFuzzerService could not start the fuzzer: %s", err.Error()))
		return
	}

	ticker := time.NewTicker(time.Second)
	for {
		select {
		case <-s.stop:
			s.logger.Info(fmt.Sprintf("GoFuzzerService spinning down fuzzer"))
			ticker.Stop()
			err = fuzzCluster.Kill()
			if err != nil {
				s.logger.Error(fmt.Sprintf("GoFuzzerService could not spin down the fuzzer: %s", err.Error()))
			}
			return
		case <-ticker.C:
			clusterState, err = fuzzCluster.State()
			if err != nil {
				s.logger.Error(fmt.Sprintf("GoFuzzerService could not start the fuzzer: %s", err.Error()))
				return
			}
			if !clusterState.Running() {
				s.logger.Error(
					fmt.Sprintf(
						"GoFuzzerService fuzz cluster stopped unexpectedly\nErrors: %s\nExit code: %v",
						err.Error(), clusterState.ExitCode()))
				return
			}
		}
	}
}
