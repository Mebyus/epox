package http

import (
	"encoding/json"
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

	task.Title = title
	task.TimeLimit = limit
	return nil
}
