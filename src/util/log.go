package util

import (
	"github.com/nullseed/logruseq"
	"github.com/sirupsen/logrus"
	"os"
)

var logger logrus.Logger

func InitLogger(config *Config) {
	seqHook := logruseq.NewSeqHook(config.SeqUrl.Value, logruseq.OptionAPIKey(config.SeqToken.Value))

	logger = logrus.Logger{
		Out:   os.Stdout,
		Hooks: make(logrus.LevelHooks),
		Level: logrus.DebugLevel,
	}

	logger.AddHook(seqHook)

	if config.Environment.Value == "production" {
		logger.Formatter = &logrus.JSONFormatter{}
	} else {
		logger.Formatter = &logrus.TextFormatter{
			ForceColors:      true,
			FullTimestamp:    false,
			QuoteEmptyFields: true,
		}
	}
}

func GetLogger() *logrus.Logger {
	return &logger
}
