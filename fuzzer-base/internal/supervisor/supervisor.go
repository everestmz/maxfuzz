package supervisor

import (
	"time"

	"maxfuzz/fuzzer-base/internal/helpers"

	"github.com/thejerf/suture"
)

func New(name string) *suture.Supervisor {
	spec := suture.Spec{
		Log: func(i string) {
			helpers.QuickLog(helpers.BasicLogger(), i)
		},
		FailureDecay:     30,               // 30 second decay
		FailureThreshold: 5,                // 5 failures
		FailureBackoff:   15 * time.Second, // Wait for 15 seconds after threshold hit
		Timeout:          30 * time.Second, // 30 seconds for service to terminate
	}

	return suture.New(name, spec)
}
