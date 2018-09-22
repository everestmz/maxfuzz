package supervisor

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/everestmz/maxfuzz/internal/constants"
	"github.com/everestmz/maxfuzz/internal/helpers"
	"github.com/everestmz/maxfuzz/internal/logging"
)

type AFLStatsService struct {
	logger logging.Logger
	stop   chan bool
	stats  chan *TargetStats
	target string
}

func NewAFLStatsService(target string, l logging.Logger, statsChan chan *TargetStats) AFLStatsService {
	return AFLStatsService{
		logger: l,
		stop:   make(chan bool),
		target: target,
		stats:  statsChan,
	}
}

func (s AFLStatsService) Stop() {
	s.logger.Info("AFLStatsService stopping")
	s.stop <- true
}

func (s AFLStatsService) Serve() {
	s.logger.Info("AFLStatsService starting")
	statsFile := filepath.Join(constants.LocalSyncDirectory, s.target, "fuzzer_stats")

	s.logger.Info("AFLStatsService waiting for fuzzer to initialize")
	for !helpers.Exists(statsFile) {
	}

	s.logger.Info("AFLStatsService watching statistics")
	ticker := time.NewTicker(time.Minute)
	for {
		select {
		case <-s.stop:
			ticker.Stop()
			return
		case <-ticker.C:
			statsMap := map[string]string{}
			file, err := os.Open(statsFile)
			if err != nil {
				s.logger.Error(fmt.Sprintf("AFLStatsService %s", err.Error()))
				return
			}

			// This adds lines like "key : val" to statsMap[key] = val
			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				spl := strings.Split(scanner.Text(), ":")
				k := strings.TrimSpace(spl[0])
				v := strings.TrimSpace(spl[1])
				statsMap[k] = v
			}

			newStats := TargetStats{}
			newStats.ID = s.target
			execsPerSecond, err := strconv.ParseFloat(
				statsMap["execs_per_sec"],
				64,
			)
			if err != nil {
				s.logger.Error(fmt.Sprintf("AFLStatsService could not parse execs_per_sec: %s", err.Error()))
				return
			}
			newStats.TestsPerSecond = execsPerSecond
			uniqueCrashes, err := strconv.Atoi(statsMap["unique_crashes"])
			if err != nil {
				s.logger.Error(fmt.Sprintf("AFLStatsService could not parse unique_crashes: %s", err.Error()))
				return
			}

			uniqueHangs, err := strconv.Atoi(statsMap["unique_hangs"])
			if err != nil {
				s.logger.Error(fmt.Sprintf("AFLStatsService could not parse unique_hangs: %s", err.Error()))
				return
			}
			newStats.BugsFound = uniqueCrashes + uniqueHangs

			s.stats <- &newStats
			file.Close()
		}
	}
}
