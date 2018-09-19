package main

import (
	"fmt"
	"os"
	"time"

	"github.com/thejerf/suture"

	"github.com/everestmz/maxfuzz/internal/logging"
	"github.com/everestmz/maxfuzz/internal/supervisor"

	"github.com/sirupsen/logrus"
)

var log = &logrus.Logger{
	Out:       os.Stderr,
	Formatter: new(logrus.JSONFormatter),
	Hooks:     make(logrus.LevelHooks),
	Level:     logrus.DebugLevel,
}
var currentTarget string
var fuzzInterval = 7200 //seconds
var stopChan chan bool
var fuzzServices = map[string]func(string) *suture.Supervisor{
	"c":   supervisor.NewCFuzzer,
	"c++": supervisor.NewCFuzzer,
	"go":  supervisor.NewGoFuzzer,
}

func nextTarget() string {
	if len(targets) == 0 {
		return ""
	}
	targetsLock.RLock()
	if len(targets) == 1 {
		for k := range targets {
			targetsLock.RUnlock()
			return k
		}
	}
	selectedTarget := "+NOT_SELECTED+"
	previousTarget := currentTarget
	var metric int64
	for k, v := range targetsTimer {
		if selectedTarget == "+NOT_SELECTED+" {
			selectedTarget = k
			metric = v
		}
		if v < metric && k != previousTarget {
			selectedTarget = k
			metric = v
		}
	}
	targetsLock.RUnlock()

	return selectedTarget
}

func interruptTarget(t string) {
	if t == currentTarget {
		stopChan <- true
	}
}

func logMessage(msg string) *logrus.Entry {
	return log.WithFields(
		logrus.Fields{
			"message": msg,
		},
	)
}

func fuzz() {
	var timer *time.Timer
	var skipFuzzerStartup = false
	stopChan = make(chan bool)
	logMessage("Waiting for targets...").Info()

	for {
		targetsLock.RLock()
		targetCount := len(targets)
		targetsLock.RUnlock()
		switch targetCount {
		case 0:
			time.Sleep(time.Second)
		default:
			if !skipFuzzerStartup {
				// Get "next" target from list, set next to new one, set supervisor to fuzz target
				logMessage(fmt.Sprintf("Fuzzer target status: %+v", targetsTimer)).Info()
				currentTarget = nextTarget()

				t := targets[currentTarget]
				newFuzzService, ok := fuzzServices[t.Language]
				if !ok {
					logMessage(fmt.Sprintf("Invalid language %s for target %s", t.Language, t.ID)).Info()
					continue
				}
				fuzzer := newFuzzService(currentTarget)
				fuzzerSupervisor = supervisor.New(logging.NewFuzzerLogger(currentTarget), currentTarget)
				fuzzerSupervisor.Add(fuzzer)
				fuzzerSupervisor.ServeBackground()

				logMessage(fmt.Sprintf("Fuzzing new target: %s...", currentTarget)).Info()
				timer = time.NewTimer(time.Duration(fuzzInterval) * time.Second)
			}
			skipFuzzerStartup = false
			select {
			case _ = <-timer.C:
				targetsLock.Lock()
				targetsTimer[currentTarget] = time.Now().Unix()
				targetsLock.Unlock()
				if nextTarget() == currentTarget {
					timer = time.NewTimer(time.Duration(fuzzInterval) * time.Second)
					skipFuzzerStartup = true
					continue
				}
				logMessage(fmt.Sprintf("Cycle for target %s finished. Picking new target...", currentTarget)).Info()
			case _ = <-stopChan:
				logMessage(fmt.Sprintf("Target %s removed. Picking new target...", currentTarget)).Info()
			}
			logMessage(fmt.Sprintf("Killing target %s...", currentTarget)).Info()
			fuzzerSupervisor.Stop()
		}
	}
}
