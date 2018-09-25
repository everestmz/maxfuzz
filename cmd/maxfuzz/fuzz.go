package main

import (
	"fmt"
	"os"
	"time"

	"github.com/everestmz/maxfuzz/internal/logging"
	"github.com/everestmz/maxfuzz/internal/supervisor"

	"github.com/sirupsen/logrus"
	"github.com/thejerf/suture"
)

var log = &logrus.Logger{
	Out:       os.Stderr,
	Formatter: new(logrus.JSONFormatter),
	Hooks:     make(logrus.LevelHooks),
	Level:     logrus.DebugLevel,
}

////////////////////
//
// COMMON VARS
//
var statsChan chan *supervisor.TargetStats
var fuzzServices = map[string]func(string, chan *supervisor.TargetStats) *suture.Supervisor{
	"c":   supervisor.NewCFuzzer,
	"c++": supervisor.NewCFuzzer,
	"go":  supervisor.NewGoFuzzer,
}

//
///////////////////
//
// ROUND ROBIN FUZZING
//
var currentTarget string
var fuzzInterval = 7200 //seconds
var stopChan chan string

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

//
///////////////////
//
// PARALLEL FUZZING
//
var parallelFuzzers map[string]*suture.Supervisor
var parallelAddChan chan *Target

//
///////////////////
func interruptTarget(t string) {
	if fuzzStrategy == "robin" {
		if t == currentTarget {
			stopChan <- t
		}
		return
	}
	// Else parallel fuzzing
	stopChan <- t
}

func logMessage(msg string) *logrus.Entry {
	return log.WithFields(
		logrus.Fields{
			"message": msg,
		},
	)
}

func watchStats() {
	for {
		select {
		case s := <-statsChan:
			targetsLock.Lock()
			targetStats[s.ID] = s
			targetsLock.Unlock()
		}
	}
}

func fuzz() {
	stopChan = make(chan string)
	statsChan = make(chan *supervisor.TargetStats)
	go watchStats()
	if fuzzStrategy == "robin" {
		fuzzRoundRobin()
	}
	fuzzParallel()
}

func fuzzParallel() {
	logMessage("Waiting for targets...").Info()
	parallelAddChan = make(chan *Target)
	parallelFuzzers = map[string]*suture.Supervisor{}

	for {
		select {
		case t := <-parallelAddChan:
			log.Println(fmt.Sprintf("%+v", *t))
			newFuzzService, ok := fuzzServices[t.Language]
			if !ok {
				logMessage(fmt.Sprintf("Invalid language %s for target %s", t.Language, t.ID)).Info()
				continue
			}
			fuzzer := newFuzzService(t.ID, statsChan)
			fuzzerSupervisor := supervisor.New(logging.NewFuzzerLogger(t.ID), t.ID)
			fuzzerSupervisor.Add(fuzzer)
			logMessage(fmt.Sprintf("Fuzzing new target %s", t.ID)).Info()
			fuzzerSupervisor.ServeBackground()
			parallelFuzzers[t.ID] = fuzzerSupervisor
		case t := <-stopChan:
			logMessage(fmt.Sprintf("Killing target %s", t)).Info()
			sup, ok := parallelFuzzers[t]
			if !ok {
				logMessage(fmt.Sprintf("Can't stop target %s", t)).Info()
				continue
			}
			sup.Stop()
			delete(parallelFuzzers, t)
			if len(parallelFuzzers) == 0 {
				logMessage("Waiting for targets...").Info()
			}
		}
	}
}

func fuzzRoundRobin() {
	var timer *time.Timer
	var skipFuzzerStartup = false
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
				fuzzer := newFuzzService(currentTarget, statsChan)
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
