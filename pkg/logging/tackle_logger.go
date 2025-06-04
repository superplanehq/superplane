package logging

import log "github.com/sirupsen/logrus"

type TackleLogger struct {
	logger *log.Entry
}

func NewTackleLogger(logger *log.Entry) TackleLogger {
	return TackleLogger{
		logger: logger,
	}
}

func (l TackleLogger) Errorf(format string, args ...interface{}) {
	l.logger.Errorf(format, args...)
}

func (l TackleLogger) Infof(format string, args ...interface{}) {
	l.logger.Infof(format, args...)
}
