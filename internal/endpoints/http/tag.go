package http

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/mebyus/epox/internal/base"
	"github.com/mebyus/epox/internal/hits"
)

type Tag struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func (p *provider) SaveTags(c *hits.Context) error {
	var body []string
	err := c.ParseBodyJSON(&body)
	if err != nil {
		return err
	}

	list, err := validateTags(body)
	if err != nil {
		return err
	}

	tags, err := p.logic.SaveTags(c.Req.Context(), c.Log, list)
	if err != nil {
		return err
	}

	var response []Tag
	if len(tags) == 0 {
		response = make([]Tag, 0)
	} else {
		response = convertTags(tags)
	}
	c.RenderJSON(http.StatusOK, response)
	return nil
}

func convertTags(tags []base.Tag) []Tag {
	if len(tags) == 0 {
		return nil
	}

	list := make([]Tag, 0, len(tags))
	for _, t := range tags {
		list = append(list, Tag{
			Name: t.Name,
			ID:   t.ID.String(),
		})
	}
	return list
}

func convertTagsFromBody(list []Tag) ([]base.Tag, error) {
	if len(list) == 0 {
		return nil, nil
	}

	tags := make([]base.Tag, 0, len(list))
	for _, t := range list {
		idstr := strings.TrimSpace(t.ID)
		name := strings.TrimSpace(t.Name)
		if idstr == "" && name == "" {
			continue
		}

		id, err := parseOptionalTagID(idstr)
		if err != nil {
			return nil, err
		}

		tags = append(tags, base.Tag{
			ID:   id,
			Name: name,
		})

	}
	if len(tags) == 0 {
		return nil, nil
	}

	return tags, nil
}

func parseOptionalTagID(s string) (base.TagID, error) {
	if s == "" {
		return 0, nil
	}

	id, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0, ErrBadTagID
	}
	return base.TagID(id), nil
}

func validateTags(list []string) ([]string, error) {
	if len(list) == 0 {
		return nil, ErrNoTags
	}

	tags := make([]string, 0, len(list))
	for _, s := range list {
		t := strings.TrimSpace(s)
		if t == "" {
			return nil, ErrEmptyTag
		}
		if t != s {
			return nil, ErrTagSpaces
		}

		tags = append(tags, t)
	}
	return tags, nil
}
