package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/everestmz/maxfuzz/internal/docker"
	"github.com/everestmz/maxfuzz/internal/helpers"
	"github.com/everestmz/maxfuzz/internal/logging"
	"github.com/everestmz/maxfuzz/internal/supervisor"
	"github.com/everestmz/maxfuzz/internal/types"

	"github.com/gin-gonic/gin"
	"github.com/thejerf/suture"
)

type Status struct {
	State          string                    `json:"state"` //FUZZING, IDLE, or ERROR
	Message        string                    `json:"message"`
	Targets        []*supervisor.TargetStats `json:"targets"`
	TestsPerSecond float64                   `json:"tests_per_second"`
	BugsFound      int                       `json:"bugs_found"`
}

var targets map[string]*types.Target
var targetsTimer map[string]int64
var targetStats map[string]*supervisor.TargetStats
var targetsLock sync.RWMutex
var fuzzerSupervisor *suture.Supervisor
var fuzzStrategy string // parallel or robin

func addTarget(t *types.Target) error {
	targetsLock.Lock()
	_, exists := targets[t.UniqueID]
	if exists {

		targetsLock.Unlock()
		return fmt.Errorf(fmt.Sprintf("Target %s already exists", t.UniqueID))
	}
	targets[t.UniqueID] = t
	targetStats[t.UniqueID] = &supervisor.TargetStats{
		ID:             t.UniqueID,
		TestsPerSecond: 0,
		BugsFound:      0,
	}
	if fuzzStrategy == "robin" {
		// Round Robin fuzzing
		targetsTimer[t.UniqueID] = 0

	} else {
		// Parallel fuzzing
		parallelAddChan <- targets[t.UniqueID]
	}
	targetsLock.Unlock()
	return nil
}

func removeTarget(t *types.Target) error {
	targetsLock.Lock()
	_, exists := targets[t.UniqueID]
	if !exists {
		targetsLock.Unlock()
		return fmt.Errorf(fmt.Sprintf("Target %s does not exist", t.UniqueID))
	}
	delete(targets, t.UniqueID)
	delete(targetStats, t.UniqueID)
	// Round robin fuzzing
	if fuzzStrategy == "robin" {
		delete(targetsTimer, t.UniqueID)
	} else {
		// Parallel fuzzing
		stopChan <- t.UniqueID
	}
	interruptTarget(t.UniqueID)
	targetsLock.Unlock()
	return nil
}

func deserializeTarget(c *gin.Context) (*types.Target, error) {
	ret := types.Target{}
	b, err := c.GetRawData()
	if err != nil {
		return &ret, err
	}
	err = json.Unmarshal(b, &ret)
	if err != nil {
		return &ret, err
	}
	return &ret, nil
}

func listTargets(c *gin.Context) {
	targetsLock.RLock()
	targetArray := []*types.Target{}
	for _, v := range targets {
		targetArray = append(targetArray, v)
	}
	targetsLock.RUnlock()
	c.JSON(http.StatusOK, targetArray)
}

func registerTarget(c *gin.Context) {
	log.Info("Received register request...")
	t, err := deserializeTarget(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	log := logging.NewTargetLogger(t.Name)
	log.Info("Registering target...")
	err = addTarget(t)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}
	log.Info("Target registered")
	c.JSON(http.StatusOK, t)
}

func unregisterTarget(c *gin.Context) {
	t, err := deserializeTarget(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	log := logging.NewTargetLogger(t.Name)
	log.Info("Unregistering target...")
	err = removeTarget(t)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}
	log.Info("Target unregistered")
	if len(targets) == 0 {
		log.Info("0 targets registered. Maxfuzz on standby...")
	}
	c.JSON(http.StatusOK, t)
}

func status(c *gin.Context) {
	status := Status{}
	if len(targets) > 0 {
		status.State = "FUZZING"
	} else {
		status.State = "IDLE"
	}
	targetsLock.RLock()
	status.Targets = []*supervisor.TargetStats{}
	for _, t := range targetStats {
		status.Targets = append(status.Targets, t)
		status.BugsFound += t.BugsFound
		status.TestsPerSecond += t.TestsPerSecond
	}
	targetsLock.RUnlock()
	c.JSON(http.StatusOK, status)
}

func main() {
	// TODO: add command line params for specifying directories
	targetsLock = sync.RWMutex{}
	targets = map[string]*types.Target{}
	targetsTimer = map[string]int64{}
	targetStats = map[string]*supervisor.TargetStats{}
	maxfuzzOptions := helpers.MaxfuzzOptions()
	fuzzStrategy = maxfuzzOptions["strategy"]
	if fuzzStrategy != "robin" && fuzzStrategy != "parallel" {
		panic("Unsupported fuzz strategy!")
	}
	err := docker.Init()
	if err != nil {
		panic(err)
	}

	fuzzerLogger := logging.NewFuzzerLogger("")
	fuzzerSupervisor = supervisor.New(fuzzerLogger, "maxfuzz")

	go fuzz()

	router := gin.Default()
	router.GET("/targets", listTargets)
	router.GET("/status", status)
	router.POST("/registerTarget", registerTarget)
	router.POST("/unregisterTarget", unregisterTarget)
	router.Run(":8080")
}
