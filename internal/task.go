package internal

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"github.com/csr-ugra/avito-estate-parser/internal/db"
	"github.com/uptrace/bun"
	"time"
)

type parsingTaskLocation struct {
	Id   int
	Name string
}

type parsingTaskTarget struct {
	Id            int
	Name          string
	FilterText    string
	SubfilterText string
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

	url, err := buildUrl(location, target)
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

func buildUrl(location *db.EstateLocationModel, target *db.EstateTargetModel) (url string, err error) {
	const urlFormat = "https://www.avito.ru/%s/%s"

	if location.UrlPart == "" {
		return "", fmt.Errorf("location model does not have a url part")
	}

	if target.UrlPart == "" {
		return "", fmt.Errorf("target model does not have a url part")
	}

	url = fmt.Sprintf(urlFormat, location.UrlPart, target.UrlPart)

	return url, nil
}

func LoadTasks(ctx context.Context, connection bun.IDB) (tasks []*ParsingTask, err error) {
	locations, err := db.GetLocations(ctx, connection)
	if err != nil {
		return nil, err
	}
	if len(locations) == 0 {
		return nil, errors.New("no locations specified")
	}

	targets, err := db.GetTargets(ctx, connection)
	if err != nil {
		return nil, err
	}
	if len(targets) == 0 {
		return nil, errors.New("no targets specified")
	}

	taskList, err := db.GetTasks(ctx, connection)
	if err != nil {
		return nil, err
	}
	if len(taskList) == 0 {
		return nil, errors.New("no tasks specified")
	}

	tasks = make([]*ParsingTask, 0, len(taskList))
	for _, task := range taskList {
		//dateEnd := time.Now().Add(48 * time.Hour)

		var dateStart time.Time
		dateStartFlag := flag.Lookup("date-start")
		if dateStartFlag != nil {
			if dateStartFlag.Value.String() != "" {
				dateStart, err = time.Parse(time.DateOnly, dateStartFlag.Value.String())
				if err != nil {
					return nil, err
				}
			} else {
				dateStart, err = time.Parse(time.DateOnly, dateStartFlag.DefValue)
				if err != nil {
					return nil, err
				}
			}
		}

		var dateEnd time.Time
		dateEndFlag := flag.Lookup("date-end")
		if dateEndFlag != nil {
			if dateEndFlag.Value.String() != "" {
				dateEnd, err = time.Parse(time.DateOnly, dateEndFlag.Value.String())
				if err != nil {
					return nil, err
				}
			} else {
				dateEnd = dateStart.Add(24 * time.Hour)
			}
		}

		t, err := NewParsingTask(task, locations, targets, dateStart, dateEnd)
		if err != nil {
			return nil, fmt.Errorf("error creating parsing task: %v", err)
		}

		tasks = append(tasks, t)
	}

	return tasks, nil
}

func SaveTaskResults(ctx context.Context, connection bun.IDB, results []*ParsingTaskResult) (int, error) {
	models := make([]*db.EstateParsingValueModel, 0, len(results))
	for _, result := range results {
		models = append(models, &db.EstateParsingValueModel{
			TaskId:           result.Task.Id,
			DateStart:        result.Task.DateStart,
			DateEnd:          result.Task.DateEnd,
			EstateTotalCount: result.EstateTotalCount,
			EstateFreeCount:  result.EstateFreeCount,
		})
	}

	insertedCount, err := db.SaveValues(ctx, connection, models)
	if err != nil {
		return 0, fmt.Errorf("error savings task results: %v", err)
	}

	return insertedCount, nil
}
