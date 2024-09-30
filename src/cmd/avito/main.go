package main

import (
	"context"
	"github.com/csr-ugra/avito-estate-parser/src/db"
	"github.com/csr-ugra/avito-estate-parser/src/parser"
	"github.com/csr-ugra/avito-estate-parser/src/util"

	"log"
	"os"
)

func main() {
	config := util.GetConfig()

	util.InitLogger(config)

	connection, err := db.GetConnection(config)
	if err != nil {
		log.Fatalln(err)
	}

	ctx := context.Background()
	parser.Start(ctx, connection, config)

	os.Exit(0)
}
