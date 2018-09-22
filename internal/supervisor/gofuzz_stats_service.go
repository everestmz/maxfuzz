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
	"github.com/everestmz/maxfuzz/internal/logging"
	"github.com/gin-contrib/sse"
)

type GofuzzStatsService struct {
	logger logging.Logger
	stop   chan bool
	stats  chan *TargetStats
	target string
}

func NewGofuzzStatsService(target string, l logging.Logger, statsChan chan *TargetStats) GofuzzStatsService {
	return GofuzzStatsService{
		logger: l,
		stop:   make(chan bool),
		target: target,
		stats:  statsChan,
	}
}

func (s GofuzzStatsService) Stop() {
	s.logger.Info("GofuzzStatsService stopping")
	s.stop <- true
}

func (s GofuzzStatsService) Serve() {
	s.logger.Info("GofuzzStatsService starting")
	statsFile := filepath.Join(constants.LocalSyncDirectory, s.target, "fuzzer_stats")

	s.logger.Info("GofuzzStatsService waiting for fuzzer to initialize")
	evCh := make(chan *sse.Event)
	go sse.Notify("http://localhost:8000/eventsource", evCh)

	s.logger.Info("GofuzzStatsService watching statistics")
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
				s.logger.Error(fmt.Sprintf("GofuzzStatsService %s", err.Error()))
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
				s.logger.Error(fmt.Sprintf("GofuzzStatsService could not parse execs_per_sec: %s", err.Error()))
				return
			}
			newStats.TestsPerSecond = execsPerSecond
			uniqueCrashes, err := strconv.Atoi(statsMap["unique_crashes"])
			if err != nil {
				s.logger.Error(fmt.Sprintf("GofuzzStatsService could not parse unique_crashes: %s", err.Error()))
				return
			}

			uniqueHangs, err := strconv.Atoi(statsMap["unique_hangs"])
			if err != nil {
				s.logger.Error(fmt.Sprintf("GofuzzStatsService could not parse unique_hangs: %s", err.Error()))
				return
			}
			newStats.BugsFound = uniqueCrashes + uniqueHangs

			s.stats <- &newStats
			file.Close()
		}
	}
}
