package db

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"github.com/uptrace/bun/extra/bundebug"
	"os"
)

const connectionStringEnvName = "DB_CONNECTION_STRING"

func getConnectionString() (string, error) {
	if connectionString, ok := os.LookupEnv(connectionStringEnvName); ok {
		return connectionString, nil
	}

	return "", fmt.Errorf("make sure that env variable %s is set and in DSN format", connectionStringEnvName)
}

func GetConnection() (*bun.DB, error) {
	dsn, err := getConnectionString()
	if err != nil {
		return nil, err
	}

	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))
	db := bun.NewDB(sqldb, pgdialect.New())

	db.AddQueryHook(bundebug.NewQueryHook(
		bundebug.WithEnabled(false),

		// BUNDEBUG=1 logs failed queries
		// BUNDEBUG=2 logs all queries
		bundebug.FromEnv("BUNDEBUG")))

	if err = db.Ping(); err != nil {
		return nil, err
	}

	return db, nil
}

func GetTasks(ctx context.Context, connection bun.IDB) (tasks []*EstateParsingTaskModel, err error) {
	err = connection.NewSelect().Model(&tasks).Order("id").Scan(ctx)

	return tasks, err
}

func GetLocations(ctx context.Context, connection bun.IDB) (locations []*EstateLocationModel, err error) {
	err = connection.NewSelect().Model(&locations).Scan(ctx)

	return locations, err
}

func GetTargets(ctx context.Context, connection bun.IDB) (targets []*EstateTargetModel, err error) {
	err = connection.NewSelect().Model(&targets).Scan(ctx)

	return targets, err
}

func SaveValues(ctx context.Context, connection bun.IDB, values []*EstateParsingValueModel) error {
	if len(values) == 0 {
		return nil
	}

	_, err := connection.NewInsert().Model(&values).Exec(ctx)

	return err
}
