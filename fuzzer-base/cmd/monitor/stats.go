package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"sync"

	"maxfuzz/fuzzer-base/internal/helpers"

	"github.com/graphql-go/graphql"
)

type AFLStats struct {
	StartTime       string  `json:"startTime"`
	LastUpdate      string  `json:"lastUpdate"`
	CyclesDone      int     `json:"cyclesDone"`
	ExecsDone       int     `json:"execsDone"`
	ExecsPerSecond  float32 `json:"execsPerSecond"`
	TotalPaths      int     `json:"totalPaths"`
	FavoredPaths    int     `json:"favoredPaths"`
	PathsFound      int     `json:"pathsFound"`
	PathsImported   int     `json:"pathsImported"`
	UniqueCrashes   int     `json:"uniqueCrashes"`
	UniqueHangs     int     `json:"uniqueHangs"`
	ExecsSinceCrash int     `json:"execsSinceCrash"`
	ExecTimeout     int     `json:"execTimeout"`
	Stability       float32 `json:"stability"`
	AFLBanner       string  `json:"aflBanner"`
}

var gqlStatsType = graphql.NewObject(graphql.ObjectConfig{
	Name: "Stats",
	Fields: graphql.Fields{
		"startTime": &graphql.Field{
			Type: graphql.String,
		},
		"lastUpdate": &graphql.Field{
			Type: graphql.String,
		},
		"cyclesDone": &graphql.Field{
			Type: graphql.Int,
		},
		"execsDone": &graphql.Field{
			Type: graphql.Int,
		},
		"execsPerSecond": &graphql.Field{
			Type: graphql.Float,
		},
		"totalPaths": &graphql.Field{
			Type: graphql.Int,
		},
		"favoredPaths": &graphql.Field{
			Type: graphql.Int,
		},
		"pathsFound": &graphql.Field{
			Type: graphql.Int,
		},
		"pathsImported": &graphql.Field{
			Type: graphql.Int,
		},
		"uniqueCrashes": &graphql.Field{
			Type: graphql.Int,
		},
		"uniqueHangs": &graphql.Field{
			Type: graphql.Int,
		},
		"execsSinceCrash": &graphql.Field{
			Type: graphql.Int,
		},
		"execTimeout": &graphql.Field{
			Type: graphql.Int,
		},
		"stability": &graphql.Field{
			Type: graphql.Float,
		},
		"aflBanner": &graphql.Field{
			Type: graphql.String,
		},
	},
})

var rootQuery = graphql.NewObject(graphql.ObjectConfig{
	Name: "RootQuery",
	Fields: graphql.Fields{
		"stats": &graphql.Field{
			Type:        graphql.NewList(gqlStatsType),
			Description: "Get fuzzer stats",
			Resolve:     resolveStats,
		},
	},
})

var schema, _ = graphql.NewSchema(graphql.SchemaConfig{
	Query: rootQuery,
})

var statsMutex = &sync.Mutex{}
var slaveStats = &AFLStats{}
var masterStats = &AFLStats{}

func resolveStats(params graphql.ResolveParams) (interface{}, error) {
	statsList := make([]*AFLStats, 0)
	statsMutex.Lock()
	statsList = append(statsList, masterStats, slaveStats)
	statsMutex.Unlock()
	return statsList, nil
}

func updateStats() {
	liveFuzzers, err := ioutil.ReadDir("/root/fuzz_out")
	helpers.Check("Failed to read /root/fuzz_out %v", err)

	newSlaveStats := AFLStats{}
	newMasterStats := AFLStats{}

	for _, fuzzerInstance := range liveFuzzers {
		summary_map := make(map[string]string)
		file, err := os.Open(
			fmt.Sprintf("/root/fuzz_out/%v/fuzzer_stats", fuzzerInstance.Name()),
		)
		helpers.Check("Opening fuzzer stats failed %v", err)
		defer file.Close()

		// This adds lines like "key : val" to summary_map[key] = val
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			spl := strings.Split(scanner.Text(), ":")
			k := strings.TrimSpace(spl[0])
			v := strings.TrimSpace(spl[1])
			summary_map[k] = v
		}

		stats := AFLStats{}
		stats.StartTime = summary_map["start_time"]

		stats.LastUpdate = summary_map["last_update"]

		stats.CyclesDone, err = strconv.Atoi(summary_map["cycles_done"])
		helpers.Check("Failed to parse cycles done: %v", err)

		stats.ExecsDone, err = strconv.Atoi(summary_map["execs_done"])
		helpers.Check("Failed to parse execs done: %v", err)

		execsPerSecond64, err := strconv.ParseFloat(
			summary_map["execs_per_sec"],
			32,
		)
		helpers.Check("Failed to parse execs per second: %v", err)
		stats.ExecsPerSecond = float32(execsPerSecond64)

		stats.TotalPaths, err = strconv.Atoi(summary_map["paths_total"])
		helpers.Check("Failed to parse total paths: %v", err)

		stats.FavoredPaths, err = strconv.Atoi(summary_map["paths_favored"])
		helpers.Check("Failed to parse favored paths: %v", err)

		stats.PathsFound, err = strconv.Atoi(summary_map["paths_found"])
		helpers.Check("Failed to parse paths found: %v", err)

		stats.PathsImported, err = strconv.Atoi(summary_map["paths_imported"])
		helpers.Check("Failed to parse paths imported: %v", err)

		stats.UniqueCrashes, err = strconv.Atoi(summary_map["unique_crashes"])
		helpers.Check("Failed to parse unique crashes: %v", err)

		stats.UniqueHangs, err = strconv.Atoi(summary_map["unique_hangs"])
		helpers.Check("Failed to parse unique hangs: %v", err)

		stats.ExecsSinceCrash, err = strconv.Atoi(summary_map["execs_since_crash"])
		helpers.Check("Failed to parse execs since crash: %v", err)

		stats.ExecTimeout, err = strconv.Atoi(summary_map["exec_timeout"])
		helpers.Check("Failed to parse exec timeout: %v", err)

		stability64, err := strconv.ParseFloat(strings.Split(summary_map["stability"], "%")[0], 32)
		helpers.Check("Failed to parse stability: %v", err)
		stats.Stability = float32(stability64)

		stats.AFLBanner = summary_map["afl_banner"]

		if fuzzerInstance.Name() == "master" {
			newMasterStats = stats
		} else {
			newSlaveStats = stats
		}
	}

	statsMutex.Lock()
	slaveStats = &newSlaveStats
	masterStats = &newMasterStats
	statsMutex.Unlock()
}
