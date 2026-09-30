package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	ep "github.com/mebyus/epox/internal/endpoints/http"
)

func main() {
	var configpath string

	flag.StringVar(&configpath, "c", "client.json", "path to config file")

	flag.Usage = func() {
		fmt.Fprint(os.Stderr, help)
		flag.PrintDefaults()
	}
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "command not specified")
		os.Exit(1)
	}
	cname := args[0]

	var config Config
	err := loadConfig(&config, configpath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load config: %v\n", err)
		os.Exit(1)
	}

	err = run(&config, cname)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func loadConfig(config *Config, path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	dec := json.NewDecoder(file)
	err = dec.Decode(config)
	if err != nil {
		return err
	}

	config.API = strings.TrimSpace(config.API)
	if config.API == "" {
		return errors.New("server api url not specified")
	}

	config.Login = strings.TrimSpace(config.Login)
	if config.Login == "" {
		return errors.New("empty login")
	}

	config.Password = strings.TrimSpace(config.Password)
	if config.Password == "" {
		return errors.New("empty password")
	}

	return nil
}

type Config struct {
	// server api url
	API string `json:"api"`

	Login    string `json:"login"`
	Password string `json:"password"`
}

func run(config *Config, cname string) error {
	hc := http.Client{Timeout: 1 * time.Second}
	token, err := login(&hc, config)
	if err != nil {
		return fmt.Errorf("do login: %v", err)
	}
	fmt.Println(token)

	switch cname {
	case "tasks/active":
		return doActiveTasks(config, &hc, token)
	case "tasks/history":
		return nil
	default:
		return fmt.Errorf("unknown command \"%s\"", cname)
	}
}

// returns auth token
func login(hc *http.Client, config *Config) (string, error) {
	data := ep.LoginData{
		Login:    config.Login,
		Password: config.Password,
	}
	body, err := json.Marshal(&data)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(context.TODO(), "POST", config.API+"/login", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	resp, err := hc.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("bad response status: %d %s", resp.StatusCode, resp.Status)
	}

	cookies := resp.Cookies()
	for _, c := range cookies {
		if c.Name == "auth" {
			return c.Value, nil
		}
	}
	return "", errors.New("auth cookie not found")
}

func doActiveTasks(config *Config, hc *http.Client, token string) error {
	req, err := http.NewRequestWithContext(context.TODO(), "GET", config.API+"/tasks/active", nil)
	if err != nil {
		return err
	}
	req.AddCookie(&http.Cookie{
		Name:  "auth",
		Value: token,
	})

	resp, err := hc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad response status: %d %s", resp.StatusCode, resp.Status)
	}

	var tasks []ep.ActiveTask
	dec := json.NewDecoder(resp.Body)
	err = dec.Decode(&tasks)
	if err != nil {
		return err
	}

	printActiveTasks(tasks)
	return nil
}

func printActiveTasks(tasks []ep.ActiveTask) {
	printActiveTask(&tasks[0])
	for _, t := range tasks[1:] {
		fmt.Printf("\n====================================\n\n")
		printActiveTask(&t)
	}
}

func printActiveTask(task *ep.ActiveTask) {
	fmt.Printf("(%s) %s\n", task.ID, task.Title)

	if len(task.Tags) != 0 {
		list := make([]string, 0, len(task.Tags))
		for _, t := range task.Tags {
			list = append(list, "#"+t.Name)
		}
		fmt.Printf("%s\n", strings.Join(list, " "))
	}
	
	if task.Deadline != "" {
		fmt.Printf("\ndeadline: %s\n", task.Deadline)
	}
}
