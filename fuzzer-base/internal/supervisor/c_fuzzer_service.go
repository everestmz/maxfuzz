package supervisor

import (
	"fmt"
	"os"

	"github.com/everestmz/maxfuzz/fuzzer-base/internal/helpers"

	"github.com/everestmz/maxfuzz/fuzzer-base/internal/constants"
	"github.com/everestmz/maxfuzz/fuzzer-base/internal/logging"
	"github.com/everestmz/maxfuzz/fuzzer-base/internal/storage"

	"github.com/go-cmd/cmd"
	"github.com/subosito/gotenv"
	"github.com/thejerf/suture"
)

type CFuzzerService struct {
	logger   logging.Logger
	targetID string
	stop     chan bool
}

var aflCmdOptions = cmd.Options{
	Buffered:  false,
	Streaming: true,
}

func NewCFuzzer(target string) *suture.Supervisor {
	log := logging.NewTargetLogger(target)
	ret := New(log, target)
	ret.Add(NewBackupService(target, log))
	ret.Add(NewAFLCrashService(target, log))
	ret.Add(CFuzzerService{
		logging.NewTargetLogger(target),
		target,
		make(chan bool),
	})
	return ret
}

func (s CFuzzerService) Stop() {
	s.logger.Info(fmt.Sprintf("CFuzzerService stopping"))
	s.stop <- true
	os.Remove(constants.FuzzerLocation)
}

func (s CFuzzerService) Serve() {
	s.logger.Info(fmt.Sprintf("CFuzzerService starting"))
	preFuzzCleanup()
	storageHandler, err := storage.Init(s.targetID)
	if err != nil {
		s.logger.Error(fmt.Sprintf("CFuzzerService could not initialize storageHandler: %s", err.Error()))
		return
	}

	// Pre-run sync and download steps
	s.logger.Info(fmt.Sprintf("CFuzzerService setting up target"))
	err = initialFuzzerSetup(s.logger, storageHandler)
	if err != nil {
		s.logger.Error(fmt.Sprintf("CFuzzerService could not initialize fuzzer: %s", err.Error()))
		return
	}

	// Ensure that we have the correct environment populated
	s.logger.Info(fmt.Sprintf("CFuzzerService populating environment"))
	env, err := os.Open(constants.FuzzerEnvironment)
	if err != nil {
		s.logger.Error("CFuzzerService could not open environnment file")
		return
	}

	pairs := gotenv.Parse(env)
	for k, v := range pairs {
		os.Setenv(k, v)
	}

	// Run the build steps
	s.logger.Info(fmt.Sprintf("CFuzzerService running build steps"))
	command := cmd.NewCmdOptions(aflCmdOptions, constants.FuzzerBuildSteps)
	stop := make(chan bool)
	go commandLogger(command, stop)

	command.Start()
	status := command.Status()
	for status.StopTs == 0 {
		status = command.Status()
	}

	stop <- true
	if status.Error != nil {
		s.logger.Error(fmt.Sprintf("CFuzzerService build steps failed with error: %s", status.Error.Error()))
		return
	}
	if status.Exit != 0 {
		s.logger.Error(fmt.Sprintf("Build steps stopped with exit code %v", status.Exit))
		return
	}

	// Finally, run the fuzzer
	s.logger.Info(fmt.Sprintf("CFuzzerService running fuzzer"))
	command = setupAFLCmd()
	stop = make(chan bool)
	opts := helpers.MaxfuzzOptions()
	if opts["suppressFuzzerOutput"] != "1" {
		go commandLogger(command, stop)
	}
	command.Start()
	status = command.Status()
	for status.StopTs == 0 {
		status = command.Status()
		if len(s.stop) > 0 {
			<-s.stop
			s.logger.Info(fmt.Sprintf("CFuzzerService spinning down fuzzer"))
			return
		}
	}

	if status.Error != nil {
		s.logger.Error(fmt.Sprintf("CFuzzerService fuzzing failed with error: %s", status.Error.Error()))
		return
	}
	if status.Exit != 0 {
		s.logger.Error(fmt.Sprintf("CFuzzerService fuzzer stopped with exit code %v", status.Exit))
		return
	}
}
