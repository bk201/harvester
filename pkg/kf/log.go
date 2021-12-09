package kf

import "github.com/sirupsen/logrus"

func Debugf(format string, args ...interface{}) {
	if logrus.IsLevelEnabled(logrus.DebugLevel) {
		logrus.Debugf("kf: "+format, args...)
	}
}

func Debug(args ...interface{}) {
	prefixed := []interface{}{"kf: "}
	logrus.Debug(append(prefixed, args...)...)
}
