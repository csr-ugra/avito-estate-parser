package parser

import (
	"fmt"
	"github.com/csr-ugra/avito-estate-parser/src/db"
	"time"
)

type parsingTaskLocation struct {
	Id   int
	Name string
}

type parsingTaskTarget struct {
	Id   int
	Name string
}

type ParsingTask struct {
	Id            int
	Location      *parsingTaskLocation
	Target        *parsingTaskTarget
	Description   string
	ValidateTitle string
	Url           string
	DateStart     *time.Time
	DateEnd       *time.Time
}

type ParsingTaskResult struct {
	Task             *ParsingTask
	EstateTotalCount int
	EstateFreeCount  int
}

func getLocationById(locations []*db.EstateLocationModel, id int) (location *db.EstateLocationModel, exist bool) {
	if len(locations) == 0 {
		return nil, false
	}

	for _, l := range locations {
		if l.Id == id {
			location = l
			return location, true
		}
	}

	return nil, false
}

func getTargetById(targets []*db.EstateTargetModel, id int) (target *db.EstateTargetModel, exist bool) {
	if len(targets) == 0 {
		return nil, false
	}

	for _, t := range targets {
		if t.Id == id {
			target = t

			return target, true
		}
	}

	return nil, false
}

func NewParsingTask(task *db.EstateParsingTaskModel, locations []*db.EstateLocationModel, targets []*db.EstateTargetModel, dateStart time.Time, dateEnd time.Time) (*ParsingTask, error) {
	location, ok := getLocationById(locations, task.EstateLocationId)
	if !ok {
		return nil, fmt.Errorf("location with id %d not found", task.EstateLocationId)
	}

	target, ok := getTargetById(targets, task.EstateTargetId)
	if !ok {
		return nil, fmt.Errorf("target with id %d not found", task.EstateTargetId)
	}

	url, err := BuildUrl(location, target)
	if err != nil {
		return nil, err
	}

	return &ParsingTask{
		Id: task.Id,
		Location: &parsingTaskLocation{
			Id:   location.Id,
			Name: location.Name,
		},
		Target: &parsingTaskTarget{
			Id:   target.Id,
			Name: target.Name,
		},
		Description:   task.Description,
		ValidateTitle: task.ValidateTitle,
		Url:           url,
		DateStart:     &dateStart,
		DateEnd:       &dateEnd,
	}, nil
}
