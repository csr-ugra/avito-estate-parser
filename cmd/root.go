package cmd

import (
	"context"
	"flag"
	"github.com/csr-ugra/avito-estate-parser/internal"
	"github.com/csr-ugra/avito-estate-parser/internal/log"
	"github.com/csr-ugra/avito-estate-parser/internal/parser_rod"
	"github.com/csr-ugra/avito-estate-parser/internal/util"
	"github.com/uptrace/bun"
)

func Run(ctx context.Context, connection bun.IDB, config *util.Config) error {
	var dryRun bool
	flag.BoolVar(&dryRun, "dry", false, "dry run")
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
	results, err = parser_rod.Start(ctx, config, tasks)

	if err != nil {
		return err
	}

	if dryRun {
		logger.Debug("saving parsing results to db")
	} else {
		err = internal.SaveTaskResults(ctx, connection, results)
		if err != nil {
			return err
		}
		logger.WithField("ResultCount", len(results)).Info("saved parsing results to db")
	}

	return nil
}
