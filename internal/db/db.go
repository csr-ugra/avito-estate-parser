package db

import (
	"context"
	"database/sql"
	"github.com/csr-ugra/avito-estate-parser/internal/util"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"github.com/uptrace/bun/extra/bundebug"
)

func GetConnection(config *util.Config) (*bun.DB, error) {
	sqlDb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(config.DbConnectionString.Value)))
	db := bun.NewDB(sqlDb, pgdialect.New())

	db.AddQueryHook(bundebug.NewQueryHook(
		bundebug.WithEnabled(false),

		// BUNDEBUG=1 logs failed queries
		// BUNDEBUG=2 logs all queries
		bundebug.FromEnv("BUNDEBUG")))

	if err := db.Ping(); err != nil {
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

func SaveValues(ctx context.Context, connection bun.IDB, values []*EstateParsingValueModel) (affectedCount int, err error) {
	if len(values) == 0 {
		return 0, nil
	}

	res, err := connection.NewInsert().Model(&values).On("CONFLICT DO NOTHING").Exec(ctx)
	if err != nil {
		return 0, err
	}

	c, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}

	return int(c), err
}
