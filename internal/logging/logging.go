package logging

import (
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
)

type Logger interface {
	Info(string)
	Error(string)
}

type TargetLogger struct {
	TargetID string
	logger   *logrus.Logger
}

func NewTargetLogger(targetID string) *TargetLogger {
	logger := &logrus.Logger{
		Out:       os.Stderr,
		Formatter: new(logrus.JSONFormatter),
		Hooks:     make(logrus.LevelHooks),
		Level:     logrus.DebugLevel,
	}

	toReturn := TargetLogger{
		TargetID: targetID,
		logger:   logger,
	}
	return &toReturn
}

func (l *TargetLogger) wrapMessage(msg string) string {
	return fmt.Sprintf("[target:%s] %s", l.TargetID, msg)
}

func (l *TargetLogger) Info(msg string) {
	l.logger.WithFields(
		logrus.Fields{
			"message": l.wrapMessage(msg),
			"target":  l.TargetID,
		},
	).Info()
}

func (l *TargetLogger) Error(msg string) {
	l.logger.WithFields(
		logrus.Fields{
			"message": l.wrapMessage(msg),
			"target":  l.TargetID,
		},
	).Error()
}

type FuzzerLogger struct {
	logger *logrus.Logger
	Fuzzer string
}

func NewFuzzerLogger(fuzzer string) *FuzzerLogger {
	logger := &logrus.Logger{
		Out:       os.Stderr,
		Formatter: new(logrus.JSONFormatter),
		Hooks:     make(logrus.LevelHooks),
		Level:     logrus.DebugLevel,
	}

	toReturn := FuzzerLogger{
		logger: logger,
		Fuzzer: fuzzer,
	}
	return &toReturn
}

func (l *FuzzerLogger) wrapMessage(msg string) string {
	return fmt.Sprintf("[fuzzer:%s] %s", l.Fuzzer, msg)
}

func (l *FuzzerLogger) Info(msg string) {
	l.logger.WithFields(
		logrus.Fields{
			"message": l.wrapMessage(msg),
			"target":  l.Fuzzer,
		},
	).Info()
}

func (l *FuzzerLogger) Error(msg string) {
	l.logger.WithFields(
		logrus.Fields{
			"message": l.wrapMessage(msg),
			"target":  l.Fuzzer,
		},
	).Error()
}
