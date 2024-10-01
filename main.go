package main

import (
	"context"
	"github.com/csr-ugra/avito-estate-parser/cmd"
	"github.com/csr-ugra/avito-estate-parser/internal/db"
	"github.com/csr-ugra/avito-estate-parser/internal/log"
	"github.com/csr-ugra/avito-estate-parser/internal/util"
	"os"
)

func main() {
	config := util.GetConfig()

	log.InitLogger(config)

	// log panic error
	defer func() {
		if r := recover(); r != nil {
			logger := log.GetLogger()
			logger.Panic(r)
		}
	}()

	connection, err := db.GetConnection(config)
	if err != nil {
		// re-fetching logger to log with all fields appended during program run
		logger := log.GetLogger()
		logger.Fatalln(err)
	}

	ctx := context.Background()

	err = cmd.Run(ctx, connection, config)
	if err != nil {
		logger := log.GetLogger()
		logger.Fatalln(err)
	}

	os.Exit(0)
}
