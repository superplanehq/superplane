package logging

import log "github.com/sirupsen/logrus"

type QuietPublisherLogger struct{}

func NewQuietPublisherLogger() QuietPublisherLogger {
	return QuietPublisherLogger{}
}

func (QuietPublisherLogger) Infof(string, ...interface{}) {}

func (QuietPublisherLogger) Errorf(format string, args ...interface{}) {
	log.Errorf(format, args...)
}
