package http

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/mebyus/epox/internal/base"
	"github.com/mebyus/epox/internal/hits"
)

type RepTask struct {
	Tags []Tag `json:"tags,omitempty"`

	Schedule    json.RawMessage `json:"schedule"`
	NextTrigger string          `json:"next_trigger"`
	Timezone    string          `json:"timezone"`

	CreateTime string `json:"create_time"`

	TimeLimit string `json:"time_limit,omitempty"`
	Type      string `json:"type"`

	Title       string `json:"title"`
	Description string `json:"description,omitempty"`

	ID string `json:"id"`

	Importance  uint32 `json:"importance,omitempty"`
	MaxProgress uint32 `json:"max_progress,omitempty"`
}

func (p *provider) AddRepTask(c *hits.Context) error {
	userID := getUserID(c)

	var body RepTask
	err := c.ParseBodyJSON(&body)
	if err != nil {
		return err
	}

	task := base.RepTask{UserID: userID}
	err = validateRepTask(&body, &task)
	if err != nil {
		return err
	}

	err = p.logic.AddRepTask(c.Req.Context(), c.Log, &task)
	if err != nil {
		return err
	}

	resp := RespID{ID: task.ID.String()}
	c.RenderJSON(http.StatusOK, &resp)
	return nil
}

func validateRepTask(body *RepTask, task *base.RepTask) error {
	var err error

	title := strings.TrimSpace(body.Title)
	if title == "" {
		return ErrEmptyTitle
	}

	var limit time.Duration
	limitstr := strings.TrimSpace(body.TimeLimit)
	if limitstr != "" {
		limit, err = parseDuration(limitstr)
		if err != nil {
			return err
		}
	}

	tags, err := convertTagsFromBody(body.Tags)
	if err != nil {
		return err
	}

	schedule, err := parseRepSchedule(body.Type, body.Schedule, body.NextTrigger, body.Timezone)
	if err != nil {
		return err
	}

	task.Title = title
	task.Tags = tags
	task.Schedule = schedule
	task.Timezone = schedule.Timezone()
	task.NextTrigger = base.MicroTime(schedule.Next(time.Time{}).UnixMicro())
	task.Description = strings.TrimSpace(body.Description)
	task.TimeLimit = limit
	task.MaxProgress = body.MaxProgress
	task.Importance = body.Importance
	return nil
}

func parseRepSchedule(typ string, data []byte, nextstr, tzstr string) (base.RepSchedule, error) {
	if len(data) == 0 {
		return nil, errors.New("empty schedule")
	}

	nextstr = strings.TrimSpace(nextstr)
	tzstr = strings.TrimSpace(tzstr)
	typ = strings.TrimSpace(typ)
	switch typ {
	case "":
		return nil, errors.New("empty type")
	case "daily":
		return parseDailySchedule(data, nextstr, tzstr)
	case "weekly":
		panic("stub")
	default:
		return nil, fmt.Errorf("unknown type \"%s\"", typ)
	}
}

type DailySchedule struct {
	// Local clock trigger time in HH:MM or HH:MM:SS format.
	//
	// Can be omitted if next trigger time is supplied.
	Clock string `json:"clock"`

	// Zero is valid value meaning that schedule triggers every day.
	DayDelay uint32 `json:"day_delay,omitzero"`
}

func parseDailySchedule(data []byte, nextstr, tzstr string) (base.RepSchedule, error) {
	if nextstr == "" && tzstr == "" {
		return nil, errors.New("no timezone")
	}

	var info DailySchedule
	err := json.Unmarshal(data, &info)
	if err != nil {
		return nil, err
	}

	if nextstr == "" || info.Clock != "" {
		return nil, errors.New("support for clock time not implemented")
	}

	var tz *time.Location
	if tzstr != "" {
		tz, err = time.LoadLocation(tzstr)
		if err != nil {
			return nil, err
		}
	}

	next, err := parseTime(nextstr)
	if err != nil {
		return nil, err
	}
	if tz != nil {
		next = next.In(tz)
	}

	return base.NewDailySchedule(next, info.DayDelay), nil
}
