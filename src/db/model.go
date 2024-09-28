package db

import (
	"github.com/uptrace/bun"
	"time"
)

type EstateLocationModel struct {
	bun.BaseModel `bun:"table:avito_estate_locations,alias:ael"`
	Id            int    `bun:"id,pk,autoincrement"`
	Name          string `bun:"name,notnull"`
	UrlPart       string `bun:"url_part,notnull"`
}

type EstateTargetModel struct {
	bun.BaseModel `bun:"table:avito_estate_targets,alias:aet"`
	Id            int    `bun:"id,pk,autoincrement"`
	Name          string `bun:"name,notnull"`
	UrlPart       string `bun:"url_part,notnull"`
}

type EstateParsingTaskModel struct {
	bun.BaseModel    `bun:"table:avito_estate_parsing_tasks,alias:aept"`
	Id               int    `bun:"id,pk,autoincrement"`
	EstateLocationId int    `bun:"avito_estate_location_id,notnull"`
	EstateTargetId   int    `bun:"avito_estate_target_id,notnull"`
	Description      string `bun:"description,notnull"`
	ValidateTitle    string `bun:"validate_title,notnull"`
}

type EstateParsingValueModel struct {
	bun.BaseModel    `bun:"table:avito_estate_parsing_values,alias:aepv"`
	Id               int        `bun:"id,pk,autoincrement"`
	TaskId           int        `bun:"task_id,notnull"`
	DateStart        *time.Time `bun:"date_start,type:date,notnull"`
	DateEnd          *time.Time `bun:"date_end,type:date,notnull"`
	EstateTotalCount int        `bun:"estate_total_count,notnull"`
	EstateFreeCount  int        `bun:"estate_free_count,notnull"`
}
