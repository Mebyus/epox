package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/mebyus/epox/internal/dms"
	"github.com/mebyus/epox/internal/gateways/mfs"
	"go.uber.org/zap"
)

func main() {
	var dir string
	var user string
	flag.StringVar(&dir, "d", "data", "data directory")
	flag.StringVar(&user, "u", "", "user name")
	flag.Parse()
	if user == "" {
		fmt.Fprintf(os.Stderr, "user not specified\n")
		os.Exit(1)
	}

	err := run(dir, user)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

type InputState struct {
	user string

	// empty if no topic is selected
	topic string

	in *bufio.Scanner

	s *mfs.Storage
}

func run(dir, user string) error {
	s, err := mfs.New(&mfs.Config{
		DataDir: dir,
	})
	if err != nil {
		return err
	}

	state := InputState{
		user: user,
		s:    s,
		in:   bufio.NewScanner(os.Stdin),
	}
	for {
		c, err := readNextCommand(&state)
		if err != nil {
			return err
		}

		switch c := c.(type) {
		case nil, *ExitCommand:
			return s.Sync()
		case *HelpCommand:
			// stub
		case *BadCommand:
			fmt.Fprintf(os.Stdout, "%s\n", c.text)
		case *ListTopicsCommand:
		case *SelectTopicCommand:
			err = execSelectTopic(&state, c.name)
		case *AddTopicCommand:
			err = execAddTopic(&state, c.name)
		case *GetMessagesCommand:
			err = execGetMessages(&state, c)
		case *AddMessageCommand:
			err = execAddMessage(&state, c)
		default:
			fmt.Fprintf(os.Stdout, "unknown command: %#v (%T)\n", c, c)
		}

		if err != nil {
			fmt.Fprintf(os.Stdout, "error on exec: %v\n", err)
			continue
		}
	}
}

type command any

type ExitCommand struct{}

type HelpCommand struct{}

type ListTopicsCommand struct {
	prefix string
}

type SelectTopicCommand struct {
	name string
}

type AddTopicCommand struct {
	name string
}

type BadCommand struct {
	text string
}

type AddMessageCommand struct {
	data []byte
}

type GetMessagesCommand struct {
}

func readNextCommand(s *InputState) (command, error) {
	printInvite(s)
	if s.topic == "" {
		return readRootCommand(s)
	}
	return readTopicCommand(s)
}

func printInvite(s *InputState) {
	if s.topic == "" {
		fmt.Fprintf(os.Stdout, "[c] ")
		return
	}

	fmt.Fprintf(os.Stdout, "* %s\n[c] ", s.topic)
}

func readTopicCommand(s *InputState) (command, error) {
	for s.in.Scan() {
		line := strings.TrimSpace(s.in.Text())
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "get") {
			return &GetMessagesCommand{}, nil
		}

		return &BadCommand{
			text: fmt.Sprintf("unknown command \"%s\"", line),
		}, nil
	}

	return nil, nil
}

func readRootCommand(s *InputState) (command, error) {
	for s.in.Scan() {
		line := strings.TrimSpace(s.in.Text())
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "select") {
			name := strings.TrimSpace(strings.TrimPrefix(line, "select"))
			if name == "" {
				return &BadCommand{
					text: "empty name in select topic command",
				}, nil
			}
			return &SelectTopicCommand{
				name: name,
			}, nil
		}

		if strings.HasPrefix(line, "add") {
			name := strings.TrimSpace(strings.TrimPrefix(line, "add"))
			if name == "" {
				return &BadCommand{
					text: "empty name in add topic command",
				}, nil
			}
			return &AddTopicCommand{
				name: name,
			}, nil
		}

		return &BadCommand{
			text: fmt.Sprintf("unknown command \"%s\"", line),
		}, nil
	}

	return nil, nil
}

func execSelectTopic(s *InputState, name string) error {
	topic := fmt.Sprintf("%s/%s", s.user, name)
	err := s.s.FindTopic(context.TODO(), zap.NewNop(), topic)
	if err != nil {
		return err
	}

	s.topic = name
	return nil
}

func execAddTopic(s *InputState, name string) error {
	topic := fmt.Sprintf("%s/%s", s.user, name)
	err := s.s.AddTopic(context.TODO(), zap.NewNop(), topic)
	if err != nil {
		return err
	}

	return nil
}

func execAddMessage(s *InputState, c *AddMessageCommand) error {
	return nil
}

func execGetMessages(s *InputState, c *GetMessagesCommand) error {
	return listMessages(s.s, fmt.Sprintf("%s/%s", s.user, s.topic))
}

func listMessages(s *mfs.Storage, topic string) error {
	list, err := s.List(context.TODO(), zap.NewNop(), &dms.ListOptions{
		Topic:  topic,
		Limit:  16,
		Latest: true,
	})
	if err != nil {
		return err
	}

	printMessages(list)
	return nil
}

func printMessages(list []dms.Message) {
	if len(list) == 0 {
		fmt.Fprintf(os.Stdout, "[nil]\n")
	}
	for _, msg := range list {
		if len(msg.Data) == 0 {
			continue
		}
		printMessage(&msg)
	}
}

func printMessage(msg *dms.Message) {
	fmt.Printf("[%08X]\n", msg.Offset)
	fmt.Printf("%s\n", msg.Data)
}
