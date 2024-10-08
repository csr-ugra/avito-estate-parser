package log

import (
	"github.com/csr-ugra/avito-estate-parser/internal/util"
	"github.com/google/uuid"
	"github.com/nullseed/logruseq"
	"github.com/sirupsen/logrus"
	"os"
)

var entry *logrus.Entry

type Logger = *logrus.Entry

func InitLogger(config *util.Config) {

	logger := logrus.Logger{
		Out:   os.Stdout,
		Hooks: make(logrus.LevelHooks),
		Level: logrus.DebugLevel,
	}

	if config.Environment.Value == "production" {
		logger.Formatter = &logrus.JSONFormatter{}
	} else {
		logger.Formatter = &logrus.TextFormatter{
			ForceColors:      true,
			FullTimestamp:    false,
			QuoteEmptyFields: true,
		}
	}

	if config.SeqUrl.Value != "" {
		seqHook := logruseq.NewSeqHook(config.SeqUrl.Value, logruseq.OptionAPIKey(config.SeqToken.Value))
		logger.AddHook(seqHook)
	} else {
		logger.Warn("logger running without seq hook")
	}

	u := uuid.New().String()
	entry = logger.WithField("TraceId", u)
}

func AddGlobalField(name string, value interface{}) Logger {
	entry = entry.WithField(name, value)
	return entry
}

func GetLogger() Logger {
	return entry
}
