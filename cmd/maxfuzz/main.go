package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/everestmz/maxfuzz/internal/docker"
	"github.com/everestmz/maxfuzz/internal/logging"
	"github.com/everestmz/maxfuzz/internal/supervisor"

	"github.com/gin-gonic/gin"
	"github.com/thejerf/suture"
)

type Target struct {
	ID       string `json:"id"`
	Language string `json:"language"`
	Location string `json:"location"`
}

type Status struct {
	State          string                    `json:"state"` //FUZZING, IDLE, or ERROR
	Message        string                    `json:"message"`
	Targets        []*supervisor.TargetStats `json:"targets"`
	TestsPerSecond float64                   `json:"tests_per_second"`
	BugsFound      int                       `json:"bugs_found"`
}

var targets map[string]*Target
var targetsTimer map[string]int64
var targetStats map[string]*supervisor.TargetStats
var targetsLock sync.RWMutex
var fuzzerSupervisor *suture.Supervisor

func deserializeTarget(c *gin.Context) (*Target, error) {
	ret := Target{}
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
	targetArray := []*Target{}
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
	log := logging.NewTargetLogger(t.ID)
	log.Info("Registering target...")

	targetsLock.Lock()
	_, exists := targets[t.ID]
	if exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Target %s already exists", t.ID)})
		targetsLock.Unlock()
		return
	}
	targets[t.ID] = t
	targetsTimer[t.ID] = 0
	targetStats[t.ID] = &supervisor.TargetStats{
		ID:             t.ID,
		TestsPerSecond: 0,
		BugsFound:      0,
	}
	targetsLock.Unlock()

	log.Info("Target registered")
	c.JSON(http.StatusOK, t)
}

func unregisterTarget(c *gin.Context) {
	t, err := deserializeTarget(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	log := logging.NewTargetLogger(t.ID)
	log.Info("Unregistering target...")
	targetsLock.Lock()
	_, exists := targets[t.ID]
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Target %s does not exist", t.ID)})
		targetsLock.Unlock()
		return
	}
	delete(targets, t.ID)
	delete(targetsTimer, t.ID)
	delete(targetStats, t.ID)
	targetsLock.Unlock()
	interruptTarget(t.ID)
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
	targets = map[string]*Target{}
	targetsTimer = map[string]int64{}
	targetStats = map[string]*supervisor.TargetStats{}
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
