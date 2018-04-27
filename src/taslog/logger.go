package taslog

import "github.com/cihub/seelog"

type Logger interface {

	Debugf(format string, params ...interface{})

	Infof(format string, params ...interface{})

	Warnf(format string, params ...interface{}) error

	Errorf(format string, params ...interface{}) error

	Debug(v ...interface{})

	Info(v ...interface{})

	Warn(v ...interface{}) error

	Error(v ...interface{}) error
}

type defaultLogger struct {
	logger seelog.LoggerInterface
}

func (l *defaultLogger) Debugf(format string, params ...interface{}) {
	l.logger.Debugf(format, params...)
}

func (l *defaultLogger) Infof(format string, params ...interface{}) {
	l.logger.Infof(format, params...)
}

func (l *defaultLogger) Warnf(format string, params ...interface{}) error {
	return l.logger.Warnf(format, params...)
}

func (l *defaultLogger) Errorf(format string, params ...interface{}) error {
	return l.logger.Errorf(format, params...)
}

func (l *defaultLogger) Debug(v ...interface{}) {
	l.logger.Debug(v)
}

func (l *defaultLogger) Info(v ...interface{}) {
	l.logger.Info(v)
}

func (l *defaultLogger) Warn(v ...interface{}) error {
	return l.logger.Warn(v)
}

func (l *defaultLogger) Error(v ...interface{}) error {
	return l.logger.Error(v)
}
