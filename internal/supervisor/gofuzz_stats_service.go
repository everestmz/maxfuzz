package supervisor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	sse "astuart.co/go-sse"
	"github.com/everestmz/maxfuzz/internal/logging"
)

type GoFuzzStats struct {
	Crashers int    `json:"Crashers"`
	Execs    int    `json:"Execs"`
	Uptime   string `json:"Uptime"`
}

type GofuzzStatsService struct {
	logger    logging.Logger
	stop      chan bool
	stats     chan *TargetStats
	target    string
	statsPort string
}

func NewGofuzzStatsService(target, statsPort string, l logging.Logger, statsChan chan *TargetStats) GofuzzStatsService {
	return GofuzzStatsService{
		logger:    l,
		stop:      make(chan bool),
		target:    target,
		stats:     statsChan,
		statsPort: statsPort,
	}
}

func (s GofuzzStatsService) Stop() {
	s.logger.Info("GofuzzStatsService stopping")
	s.stop <- true
}

func (s GofuzzStatsService) Serve() {
	s.logger.Info("GofuzzStatsService starting")

	s.logger.Info("GofuzzStatsService waiting for fuzzer to initialize")
	evCh := make(chan *sse.Event)
	go func() {
		for {
			time.Sleep(time.Second)
			sse.Notify(fmt.Sprintf("http://0.0.0.0:%s/eventsource", s.statsPort), evCh)
		}
	}()

	s.logger.Info("GofuzzStatsService watching statistics")
	for {
		select {
		case <-s.stop:
			return
		case evt := <-evCh:
			buf := new(bytes.Buffer)
			buf.ReadFrom(evt.Data)
			str := buf.String()
			newStats := GoFuzzStats{}
			err := json.Unmarshal([]byte(str), &newStats)
			if err != nil {
				s.logger.Error(fmt.Sprintf("GofuzzStatsService could not parse sse stream: %s", err.Error()))
				return
			}
			var secsRunning int
			if strings.Contains(newStats.Uptime, "h") {
				hrSplit := strings.Split(newStats.Uptime, "h")
				hrs, err := strconv.Atoi(hrSplit[0])
				if err != nil {
					s.logger.Error(fmt.Sprintf("Could not parse time string %s", newStats.Uptime))
					return
				}
				minSplit := strings.Split(hrSplit[1], "m")
				mins, err := strconv.Atoi(minSplit[0])
				if err != nil {
					s.logger.Error(fmt.Sprintf("Could not parse time string %s", newStats.Uptime))
					return
				}
				secsRunning = hrs*3600 + mins*60
			} else if strings.Contains(newStats.Uptime, "m") {
				minSplit := strings.Split(newStats.Uptime, "m")
				mins, err := strconv.Atoi(minSplit[0])
				if err != nil {
					s.logger.Error(fmt.Sprintf("Could not parse time string %s", newStats.Uptime))
					return
				}
				secSplit := strings.Split(minSplit[1], "s")
				secs, err := strconv.Atoi(secSplit[0])
				if err != nil {
					s.logger.Error(fmt.Sprintf("Could not parse time string %s", newStats.Uptime))
					return
				}
				secsRunning = secs + mins*60
			} else if strings.Contains(newStats.Uptime, "s") {
				secSplit := strings.Split(newStats.Uptime, "s")
				secs, err := strconv.Atoi(secSplit[0])
				if err != nil {
					s.logger.Error(fmt.Sprintf("Could not parse time string %s", newStats.Uptime))
					return
				}
				secsRunning = secs
			}
			log.Println(fmt.Sprintf("%v execs, %v secs", newStats.Execs, secsRunning))
			commonStats := TargetStats{
				ID:             s.target,
				BugsFound:      newStats.Crashers,
				TestsPerSecond: float64(newStats.Execs) / float64(secsRunning),
			}
			s.stats <- &commonStats
		}
	}
}
