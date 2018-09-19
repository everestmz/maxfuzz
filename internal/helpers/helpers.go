package helpers

import (
	"fmt"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"
)

var log = BasicLogger()
var fs = afero.NewOsFs()
var fuzzer = Getenv("FUZZER_NAME", "test")
var revision = Getenv("GIT_SHA", "no_git")

func MaxfuzzOptions() map[string]string {
	result := GetenvOrDie("MAXFUZZ_OPTIONS")
	ret := map[string]string{}
	split := strings.Split(result, ":")
	for _, v := range split {
		vals := strings.Split(v, "=")
		key := vals[0]
		val := vals[1]
		ret[key] = val
	}

	return ret
}

func Getenv(key, def string) string {
	temp := os.Getenv(key)
	if len(temp) == 0 {
		return def
	}
	return temp
}

func GetenvOrDie(key string) string {
	temp := os.Getenv(key)
	if len(temp) == 0 {
		exitErrorf("No environment variable found for %s.", key)
	}
	return temp
}

func BasicLogger() *logrus.Logger {
	logger := &logrus.Logger{
		Out:       os.Stderr,
		Formatter: new(logrus.JSONFormatter),
		Hooks:     make(logrus.LevelHooks),
		Level:     logrus.DebugLevel,
	}

	return logger
}

// Error helpers
func exitErrorf(msg string, args ...interface{}) {
	log.WithFields(
		logrus.Fields{"message": fmt.Sprintf(msg+"\n", args...)},
	).Fatal()
	os.Exit(1)
}

func Check(msg string, err error) {
	if err != nil {
		exitErrorf(msg, err)
	}
}

func Exists(filename string) bool {
	exists, err := afero.Exists(fs, filename)
	Check("File existence check fail: %v", err)
	return exists
}
