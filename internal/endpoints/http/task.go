package http

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/mebyus/epox/internal/base"
	"github.com/mebyus/epox/internal/hits"
)

func (p *provider) AddActiveTask(c *hits.Context) error {
	userID := getUserID(c)

	var body ActiveTask
	err := c.ParseBodyJSON(&body)
	if err != nil {
		return err
	}

	task := base.ActiveTask{UserID: userID}
	err = validateTask(&body, &task)
	if err != nil {
		return err
	}

	err = p.logic.AddTask(c.Req.Context(), c.Log, &task)
	if err != nil {
		return err
	}

	resp := RespID{ID: task.ID.String()}
	c.RenderJSON(http.StatusOK, &resp)
	return nil
}

type RespID struct {
	ID string `json:"id"`
}

type ActiveTask struct {
	Tags []Tag `json:"tags,omitempty"`

	CreateTime string `json:"create_time"`

	// Due duration in time.ParseDuration format.
	//
	// Can be used to set deadline based on current
	// datetime + due duration.
	DueDur string `json:"due_dur,omitempty"`

	Deadline string `json:"deadline,omitempty"`

	Title       string `json:"title"`
	Description string `json:"description,omitempty"`

	ID string `json:"id"`

	Importance  uint32 `json:"importance,omitempty"`
	Progress    uint32 `json:"progress,omitempty"`
	MaxProgress uint32 `json:"max_progress,omitempty"`
}

func (p *provider) GetActiveTasks(c *hits.Context) error {
	userID := getUserID(c)

	tasks, err := p.logic.GetActiveTasks(c.Req.Context(), c.Log, userID)
	if err != nil {
		return err
	}

	response := make([]ActiveTask, 0, len(tasks))
	for _, task := range tasks {
		response = append(response, ActiveTask{
			Tags:        convertTags(task.Tags),
			CreateTime:  task.CreateTime.Format(time.RFC3339),
			Deadline:    formatOptZeroTime(task.Deadline),
			Title:       task.Title,
			Description: task.Description,
			ID:          task.ID.String(),
			Importance:  task.Importance,
			Progress:    task.Progress,
			MaxProgress: task.MaxProgress,
		})
	}

	c.RenderJSON(http.StatusOK, response)
	return nil
}

type HistoryTask struct {
	Tags []Tag `json:"tags,omitempty"`

	CreateTime string `json:"create_time"`
	UpdateTime string `json:"update_time,omitempty"`
	Deadline   string `json:"deadline,omitempty"`

	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	State       string `json:"state"`

	ID string `json:"id"`

	Importance  uint32 `json:"importance,omitempty"`
	Progress    uint32 `json:"progress,omitempty"`
	MaxProgress uint32 `json:"max_progress,omitempty"`
}

func (p *provider) GetHistoryTasks(c *hits.Context) error {
	userID := getUserID(c)

	cutoffString := strings.TrimSpace(c.Query("cutoff"))
	if cutoffString == "" {
		return NewBadReqError("empty cutoff")
	}
	cutoff, err := parseTime(cutoffString)
	if err != nil {
		return err
	}
	if time.Since(cutoff) > 365*24*time.Hour {
		return NewBadReqError("cutoff is too old")
	}

	tasks, err := p.logic.GetHistoryTasks(c.Req.Context(), c.Log, userID, cutoff)
	if err != nil {
		return err
	}

	response := make([]HistoryTask, 0, len(tasks))
	for _, task := range tasks {
		response = append(response, HistoryTask{
			State:       task.State.String(),
			Tags:        convertTags(task.Tags),
			CreateTime:  task.CreateTime.Format(time.RFC3339),
			UpdateTime:  formatOptZeroTime(task.UpdateTime),
			Deadline:    formatOptZeroTime(task.Deadline),
			Title:       task.Title,
			Description: task.Description,
			ID:          task.ID.String(),
			Importance:  task.Importance,
			Progress:    task.Progress,
			MaxProgress: task.MaxProgress,
		})
	}

	c.RenderJSON(http.StatusOK, response)
	return nil
}

type ChangeTaskStateBody struct {
	ID    string `json:"id"`
	State string `json:"state"`
}

func (p *provider) ChangeTaskState(c *hits.Context) error {
	userID := getUserID(c)

	var body ChangeTaskStateBody
	err := c.ParseBodyJSON(&body)
	if err != nil {
		return err
	}

	req := base.RequestChangeTaskState{UserID: userID}
	err = validateChangeTaskState(&body, &req)
	if err != nil {
		return err
	}

	err = p.logic.ChangeTaskState(c.Req.Context(), c.Log, &req)
	if err != nil {
		return err
	}

	c.RenderStatus(http.StatusOK)
	return nil
}

func validateTask(body *ActiveTask, task *base.ActiveTask) error {
	var err error

	title := strings.TrimSpace(body.Title)
	if title == "" {
		return ErrEmptyTitle
	}

	var deadline time.Time
	deadstr := strings.TrimSpace(body.Deadline)
	duestr := strings.TrimSpace(body.DueDur)
	if deadstr != "" {
		deadline, err = parseTime(deadstr)
		if err != nil {
			return err
		}
	} else if duestr != "" {
		dur, err := parseDuration(duestr)
		if err != nil {
			return err
		}
		deadline = time.Now().Add(dur)
	}

	tags, err := convertTagsFromBody(body.Tags)
	if err != nil {
		return err
	}

	task.Title = title
	task.Tags = tags
	task.Description = strings.TrimSpace(body.Description)
	task.Deadline = deadline
	task.Importance = body.Importance
	task.Progress = body.Progress
	task.MaxProgress = body.MaxProgress
	return nil
}

func validateChangeTaskState(body *ChangeTaskStateBody, req *base.RequestChangeTaskState) error {
	state, err := parseTaskState(body.State)
	if err != nil {
		return err
	}
	id, err := parseTaskID(body.ID)
	if err != nil {
		return err
	}

	req.ID = id
	req.State = state
	return nil
}

func parseTaskID(s string) (base.TaskID, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, ErrEmptyTaskID
	}

	n, err := strconv.ParseUint(s, 10, 64)
	if err != nil || n == 0 {
		return 0, ErrBadTaskID
	}

	return base.TaskID(n), nil
}

func parseTaskState(s string) (base.TaskState, error) {
	s = strings.TrimSpace(s)

	switch s {
	case "":
		return 0, ErrEmptyTaskState
	case "active":
		return base.TaskActive, nil
	case "done":
		return base.TaskDone, nil
	case "canceled":
		return base.TaskCanceled, nil
	case "expired":
		return base.TaskExpired, nil
	default:
		return 0, NewBadReqError(fmt.Sprintf("unknown task state \"%s\"", s))
	}
}

func parseTime(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, nil
	}
	return time.Parse(time.RFC3339, s)
}

const fullDay = 24 * time.Hour

var (
	ErrEmptyDurationString = NewBadReqError("empty duration string")
	ErrNegativeDuration    = NewBadReqError("negative duration")
	ErrMissingDurationUnit = NewBadReqError("missing duration unit")
	ErrOrphanedDaysUnit    = NewBadReqError("orphaned days unit")
	ErrBadDurationSyntax   = NewBadReqError("bad duration syntax")
)

func parseDuration(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, " ", "")
	if s == "" {
		return 0, ErrEmptyDurationString
	}

	if s[0] == '-' {
		return 0, ErrNegativeDuration
	}
	last := s[len(s)-1]
	if '0' <= last && last <= '9' {
		return 0, ErrMissingDurationUnit
	}
	i := strings.Index(s, "d")
	if i < 0 {
		return time.ParseDuration(s)
	}
	if i == 0 {
		return 0, ErrOrphanedDaysUnit
	}
	daysPart := s[:i]
	restPart := s[i+1:]

	var dur time.Duration
	if strings.Contains(daysPart, ".") {
		days, err := strconv.ParseFloat(daysPart, 64)
		if err != nil {
			return 0, fmt.Errorf("bad days part \"%s\" in duration", daysPart)
		}
		dur = time.Duration(uint64(days * float64(fullDay)))
	} else {
		days, err := strconv.ParseUint(daysPart, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("bad days part \"%s\" in duration", daysPart)
		}
		dur = time.Duration(days) * fullDay
	}

	if restPart == "" {
		return dur, nil
	}
	if restPart[0] == '-' {
		return 0, ErrBadDurationSyntax
	}
	rest, err := time.ParseDuration(restPart)
	if err != nil {
		return 0, err
	}
	dur += rest
	return dur, nil
}

func formatOptZeroTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}
