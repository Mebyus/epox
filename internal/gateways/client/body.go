package client

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
)

type Server struct {
	URL  string
	Auth string
}

type ErrorBody struct {
	Error string `json:"error"`
}

// POST /ext/topic
type CreateTopicBody struct {
	Topic string `json:"topic"`
}

func NewCreateTopicBody(ctx context.Context, s *Server, topic string) (*http.Request, error) {
	u, err := url.Parse(s.URL)
	if err != nil {
		return nil, err
	}

	body, err := json.Marshal(&CreateTopicBody{Topic: topic})
	if err != nil {
		return nil, err
	}
	r, err := http.NewRequestWithContext(ctx, http.MethodPost, u.JoinPath("topic").String(), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	r.Header.Add("Authorization", "Bearer "+s.Auth)

	return r, nil
}

func ParseErrorBody(r io.Reader) (string, error) {
	var body ErrorBody
	decoder := json.NewDecoder(r)
	err := decoder.Decode(&body)
	if err != nil {
		return "", err
	}

	return body.Error, nil
}
