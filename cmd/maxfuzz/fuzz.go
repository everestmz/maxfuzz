package main

import (
	"fmt"
	"os"
	"time"

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
var fuzzInterval = 3600
var stopChan chan bool

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
			// Get "next" target from list, set next to new one, set supervisor to fuzz target
			// TODO: don't restart the fuzzer if we don't have any other ones to pick from
			logMessage(fmt.Sprintf("Fuzzer target status: %+v", targetsTimer)).Info()
			currentTarget = nextTarget()
			fuzzerSupervisor = supervisor.New(logging.NewFuzzerLogger(currentTarget), currentTarget)

			// TODO: Language mapping
			fuzzer := supervisor.NewCFuzzer(currentTarget)
			fuzzerSupervisor.Add(fuzzer)
			fuzzerSupervisor.ServeBackground()

			logMessage(fmt.Sprintf("Fuzzing new target: %s...", currentTarget)).Info()
			timer := time.NewTimer(time.Duration(fuzzInterval) * time.Second)
			select {
			case _ = <-timer.C:
				logMessage(fmt.Sprintf("Cycle for target %s finished. Picking new target...", currentTarget)).Info()
				targetsLock.Lock()
				targetsTimer[currentTarget] = time.Now().Unix()
				targetsLock.Unlock()
			case _ = <-stopChan:
				logMessage(fmt.Sprintf("Target %s removed. Picking new target...", currentTarget)).Info()
			}
			logMessage(fmt.Sprintf("Killing target %s...", currentTarget)).Info()
			fuzzerSupervisor.Stop()
		}
	}
}
