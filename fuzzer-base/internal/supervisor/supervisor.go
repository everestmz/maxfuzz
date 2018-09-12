package supervisor

import (
	"time"

	"github.com/everestmz/maxfuzz/fuzzer-base/internal/logging"

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
