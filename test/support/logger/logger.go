package logger

import (
	"io"

	log "github.com/sirupsen/logrus"
)

func DiscardLogger() *log.Entry {
	logger := log.New()
	logger.SetOutput(io.Discard)
	return log.NewEntry(logger)
}
