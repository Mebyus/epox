package http

import (
	"net/http"
	"strings"

	"github.com/mebyus/epox/internal/base"
	"github.com/mebyus/epox/internal/hits"
)

type Topic struct {
	CreateTime string `json:"create_time"`

	Path        string `json:"path"`
	Title       string `json:"title"`
	Description string `json:"desc,omitempty"`

	ID string `json:"id"`

	Offset uint64 `json:"offset"`
}

func (p *provider) AddNotesTopic(c *hits.Context) error {
	userID := getUserID(c)

	var body Topic
	err := c.ParseBodyJSON(&body)
	if err != nil {
		return err
	}

	topic := base.NotesTopic{UserID: userID}
	err = validateNotesTopic(&body, &topic)
	if err != nil {
		return err
	}

	err = p.logic.AddNotesTopic(c.Req.Context(), c.Log, &topic)
	if err != nil {
		return err
	}

	resp := RespID{ID: topic.ID.String()}
	c.RenderJSON(http.StatusOK, &resp)
	return nil
}

func (p *provider) AddNotesMessage(c *hits.Context) error {
	return nil
}

type RequestNotesBody struct {
	Path string `json:"path"`
}

type NotesMessage struct {
	Text string `json:"text"`

	CreateTime string `json:"create_time"`

	ID string `json:"id"`

	Offset uint64 `json:"offset"`
}

func (p *provider) GetNotesMessages(c *hits.Context) error {
	userID := getUserID(c)

	var body RequestNotesBody
	err := c.ParseBodyJSON(&body)
	if err != nil {
		return err
	}

	req := base.RequestNotes{UserID: userID}
	err = validateRequestNotes(&body, &req)
	if err != nil {
		return err
	}

	messages, err := p.logic.GetNotesMessages(c.Req.Context(), c.Log, &req)
	if err != nil {
		return err
	}

	response := make([]NotesMessage, 0, len(messages))
	for _, m := range messages {
		response = append(response, NotesMessage{
			Text:       m.Text,
			CreateTime: formatOptZeroTime(m.CreateTime),
			ID:         m.ID.String(),
			Offset:     m.Offset,
		})
	}

	c.RenderJSON(http.StatusOK, response)
	return nil
}

func validateRequestNotes(body *RequestNotesBody, req *base.RequestNotes) error {
	path := strings.TrimSpace(body.Path)
	if path == "" {
		return ErrEmptyPath
	}
	err := checkTopicPath(path)
	if err != nil {
		return err
	}

	req.Path = path
	return nil
}

func validateNotesTopic(body *Topic, topic *base.NotesTopic) error {
	path := strings.TrimSpace(body.Path)
	if path == "" {
		return ErrEmptyPath
	}
	err := checkTopicPath(path)
	if err != nil {
		return err
	}

	title := strings.TrimSpace(body.Title)
	if title == "" {
		return ErrEmptyTitle
	}

	topic.Path = path
	topic.Title = title
	topic.Description = strings.TrimSpace(body.Description)
	return nil
}

func checkTopicPath(path string) error {
	// TODO: optimize implementation to avoid allocations
	segments := strings.Split(path, "/")
	for _, s := range segments {
		err := checkTopicSegment(s)
		if err != nil {
			return err
		}
	}
	return nil
}

func checkTopicSegment(segment string) error {
	if segment == "" {
		return ErrEmptyPathSegment
	}

	// TODO: check characters in segment
	return nil
}
