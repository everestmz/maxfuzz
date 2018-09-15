package supervisor

import (
	"fmt"
	"log"
	"time"

	"github.com/everestmz/maxfuzz/internal/logging"

	"github.com/thejerf/suture"
)

func panicOnError(err error) {
	if err != nil {
		panic(err)
	}
}

func New(l logging.Logger, name string) *suture.Supervisor {
	spec := suture.Spec{
		Log:              l.Info,
		FailureDecay:     30,               // 30 second decay
		FailureThreshold: 5,                // 5 failures
		FailureBackoff:   15 * time.Second, // Wait for 15 seconds after threshold hit
		Timeout:          30 * time.Second, // 30 seconds for service to terminate
	}

	supervisor := suture.New(name, spec)
	return supervisor
}

// Log Writers
// TODO: some way of splitting logs so these ones are saved elsewhere (container vs maxfuzz logs)
type stderrWriter struct {
	containerID string
}

func (w stderrWriter) Write(p []byte) (int, error) {
	log.Print(fmt.Sprintf("[stderr]: %s", p))
	return len(p), nil
}

type stdoutWriter struct {
	containerID string
}

func (w stdoutWriter) Write(p []byte) (int, error) {
	log.Print(fmt.Sprintf("[stdout]: %s", p))
	return len(p), nil
}
