package docker

import (
	"fmt"
	"log"

	"github.com/everestmz/maxfuzz/internal/helpers"
)

var logger = helpers.BasicLogger()

type stderrWriter struct {
	containerID string
}

func (w stderrWriter) Write(p []byte) (int, error) {
	log.Print(fmt.Sprintf("[%s:stderr]: %s", w.containerID, p))
	return len(p), nil
}

type stdoutWriter struct {
	containerID string
}

func (w stdoutWriter) Write(p []byte) (int, error) {
	log.Print(fmt.Sprintf("[%s:stdout]: %s", w.containerID, p))
	return len(p), nil
}
