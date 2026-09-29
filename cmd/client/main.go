package main

import (
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"
)

type flags struct {
	url     string
	auth    string
	file    string
	message string
	topic   string
	key     string
	offset  string
	info    bool
	create  bool
}

func main() {
	var f flags

	flag.StringVar(&f.url, "u", "", "url of server api")
	flag.StringVar(&f.auth, "a", "", "auth token")
	flag.StringVar(&f.file, "f", "", "config file")
	flag.StringVar(&f.message, "m", "", "file with message text")
	flag.StringVar(&f.offset, "o", "", "message offset")
	flag.StringVar(&f.topic, "t", "", "topic name")
	flag.StringVar(&f.key, "k", "", "crypto key")
	flag.BoolVar(&f.info, "i", false, "display topic info")
	flag.BoolVar(&f.create, "c", false, "create new topic")

	flag.Usage = func() {
		fmt.Fprint(os.Stderr, help)
		flag.PrintDefaults()
	}
	flag.Parse()

	err := run(&f)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(f *flags) error {
	// TODO: read config file if needed
	if f.url == "" {
		return errors.New("empty server url")
	}
	if f.auth == "" {
		return errors.New("empty auth token")
	}

	common := CommonConfig{
		ServerURL: f.url,
		AuthToken: f.auth,
	}

	client := http.Client{Timeout: 15 * time.Second}

	if f.topic == "" {
		panic("stub")
	}

	if f.create {
		return createTopic(&CreateTopicConfig{
			CommonConfig: common,
			Topic:        f.topic,
		}, &client)
	}

	panic("stub")
}
