//   Copyright (C) 2018 TASChain
//
//   This program is free software: you can redistribute it and/or modify
//   it under the terms of the GNU General Public License as published by
//   the Free Software Foundation, either version 3 of the License, or
//   (at your option) any later version.
//
//   This program is distributed in the hope that it will be useful,
//   but WITHOUT ANY WARRANTY; without even the implied warranty of
//   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
//   GNU General Public License for more details.
//
//   You should have received a copy of the GNU General Public License
//   along with this program.  If not, see <https://www.gnu.org/licenses/>.

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
