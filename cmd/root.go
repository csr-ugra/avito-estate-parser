package cmd

import (
	"context"
	"flag"
	"github.com/csr-ugra/avito-estate-parser/internal"
	"github.com/csr-ugra/avito-estate-parser/internal/log"
	"github.com/csr-ugra/avito-estate-parser/internal/parser"
	"github.com/csr-ugra/avito-estate-parser/internal/util"
	"github.com/sirupsen/logrus"
	"github.com/uptrace/bun"
	"time"
)

func Run(ctx context.Context, connection bun.IDB, config *util.Config) error {
	var dryRun bool
	flag.BoolVar(&dryRun, "dry", false, "dry run")
	flag.String("date-start", time.Now().Add(24*time.Hour).Format(time.DateOnly), "start date, default: tomorrows date")
	flag.String("date-end", "", "end date, default: the day after 'date-start'")
	flag.Parse()

	logger := log.GetLogger()

	if dryRun {
		logger = log.AddGlobalField("DryRun", dryRun)
	}

	logger.Debug("retrieving tasks from db")
	tasks, err := internal.LoadTasks(ctx, connection)
	if err != nil {
		return err
	}
	logger.WithField("TaskCount", len(tasks)).Info("retrieved tasks from db")

	var results []*internal.ParsingTaskResult
	results, err = parser.Start(ctx, config, tasks)

	if err != nil {
		return err
	}

	logger.Debug("saving parsing results to db")
	if !dryRun {
		affectedCount, err := internal.SaveTaskResults(ctx, connection, results)
		if err != nil {
			return err
		}
		logger.WithFields(logrus.Fields{
			"ResultCount":      len(results),
			"AffectedRowCount": affectedCount,
		}).Info("saved parsing results to db")
	}

	return nil
}
