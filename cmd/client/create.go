package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/mebyus/epox/internal/gateways/client"
)

func createTopic(config *CreateTopicConfig, c *http.Client) error {
	req, err := client.NewCreateTopicBody(context.TODO(),
		&client.Server{
			URL:  config.ServerURL,
			Auth: config.AuthToken,
		},
		config.Topic,
	)
	if err != nil {
		return err
	}

	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		text, err := client.ParseErrorBody(resp.Body)
		if err != nil {
			// TODO: log this?
		}
		return fmt.Errorf("%d (%s) status code from server: %s", resp.StatusCode, http.StatusText(resp.StatusCode), text)
	}

	return nil
}
