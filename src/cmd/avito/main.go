package main

import (
	"context"
	"github.com/csr-ugra/avito-estate-parser/src/db"
	"github.com/csr-ugra/avito-estate-parser/src/parser"
	_ "github.com/joho/godotenv/autoload"
	"log"
	"os"
)

func main() {
	connection, err := db.GetConnection()
	if err != nil {
		log.Fatalln(err)
	}

	ctx := context.Background()
	parser.Start(ctx, connection)

	os.Exit(0)
}
