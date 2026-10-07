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

	"github.com/mebyus/epox/internal/base"
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

	err = run(&config, cname, args[1:])
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

	TokenFile string `json:"token_file"`
}

func run(config *Config, cname string, args []string) error {
	hc := http.Client{Timeout: 1 * time.Second}
	token, err := getAuthToken(&hc, config)
	if err != nil {
		return fmt.Errorf("do login: %v", err)
	}

	switch cname {
	case "tasks/active":
		return doActiveTasks(config, &hc, token)
	case "task/done":
		if len(args) == 0 {
			return errors.New("task id not specified")
		}
		return doTaskDone(config, &hc, token, args[0])
	case "tasks/history":
		return nil
	default:
		return fmt.Errorf("unknown command \"%s\"", cname)
	}
}

func getAuthToken(hc *http.Client, config *Config) (string, error) {
	tokenFile := strings.TrimSpace(config.TokenFile)
	if tokenFile == "" {
		return login(hc, config)
	}

	token := loadAuthToken(tokenFile)
	if token != "" {
		err := check(hc, config, token)
		if err == nil {
			// note error condition
			return token, nil
		}
		// TODO: maybe print this error in debug mode?
	}

	token, err := login(hc, config)
	if err != nil {
		return "", err
	}

	err = saveAuthToken(tokenFile, token)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[warn] save auth token: %v\n", err)
	}

	return token, nil
}

func loadAuthToken(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "[warn] load auth token: %v\n", err)
		}
		return ""
	}
	return string(data)
}

func saveAuthToken(path string, token string) error {
	return os.WriteFile(path, []byte(token), 0o644)
}

// check auth token through server request
func check(hc *http.Client, config *Config, token string) error {
	req, err := http.NewRequestWithContext(context.TODO(), "GET", config.API+"/session", nil)
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

	return nil
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

func doTaskDone(config *Config, hc *http.Client, token string, id string) error {
	body := ep.ChangeTaskStateBody{
		ID:    id,
		State: "done",
	}
	data, err := json.Marshal(&body)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(context.TODO(), "POST", config.API+"/task/state", bytes.NewReader(data))
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

	return nil
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

	var list []ep.ActiveTask
	dec := json.NewDecoder(resp.Body)
	err = dec.Decode(&list)
	if err != nil {
		return err
	}

	tasks, err := ep.ConvertActiveTasksFromBodyFormat(list)
	if err != nil {
		return err
	}

	base.SortTasksByUrgency(tasks, base.Now())
	printActiveTasks(tasks)
	return nil
}

func printActiveTasks(tasks []base.Task) {
	fmt.Println()
	for _, t := range tasks {
		printActiveTask(&t)
		fmt.Println()
	}
}

func printActiveTask(task *base.Task) {
	if task.RepTaskID != 0 {
		fmt.Printf("[%s.%s] %s\n", task.ID, task.RepTaskID, task.Title)
	} else {
		fmt.Printf("[%s] %s\n", task.ID, task.Title)
	}

	if len(task.Tags) != 0 {
		list := make([]string, 0, len(task.Tags))
		for _, t := range task.Tags {
			list = append(list, t.Name)
		}
		fmt.Printf("tags:  %s\n", strings.Join(list, " "))
	}

	const layout = "2006-01-02 15:04"

	now := base.Now()
	if task.StartTime != 0 && now < task.StartTime {
		// task not started yet
		left := time.Duration(task.StartTime-now) * time.Microsecond
		fmt.Printf("start: %s | %s\n", formatTimeLeft(left), task.StartTime.Time().Format(layout))

		if task.Deadline != 0 && task.Deadline > task.StartTime {
			leftts := task.Deadline - task.StartTime
			left := time.Duration(leftts) * time.Microsecond
			fmt.Printf("left:  %s | %s\n", formatTimeLeft(left), task.Deadline.Time().Format(layout))
		}
	} else {
		// task already started
		if task.Deadline != 0 {
			left := time.Until(task.Deadline.Time())
			fmt.Printf("left:  %s | %s\n", formatTimeLeft(left), task.Deadline.Time().Format(layout))
		}
	}
}

func formatTimeLeft(left time.Duration) string {
	if left <= 0 {
		return "<expired>"
	}
	if left < 4*time.Hour {
		hours := left.Truncate(time.Hour)
		minutes := left.Round(time.Minute) - 60*hours
		return fmt.Sprintf("%dh %dm", hours/time.Hour, minutes/time.Minute)
	}
	if left < 24*time.Hour {
		hours := left.Round(time.Hour)
		return fmt.Sprintf("%dh", hours/time.Hour)
	}

	days := left / (24 * time.Hour)
	hours := left.Truncate(time.Hour) - days*24*time.Hour
	if days < 3 {
		return fmt.Sprintf("%dd %dh", days, hours/time.Hour)
	}
	return fmt.Sprintf("%dd", days)
}
