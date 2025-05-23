package config

import (
	"borbAggregatorFRFR/internal/database"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
)

type Config struct {
	DbURL           string `json:"db_url"`
	CurrentUserName string `json:"current_user_name"`
}

type State struct {
	Db   *database.Queries
	Conf *Config
}

type Command struct {
	Name      string
	Arguments []string
}

type Commands struct {
	CommandFunc map[string]func(*State, Command) error
}

const configFileName = ".gatorconfig.json"
const dbUrl = `postgres://postgres:postgres@localhost:5432/gator`

func Read() Config {
	path := getHomePath()
	fmt.Println(path)
	file, err := os.ReadFile(path)
	if err != nil {
		fmt.Println(err)
		fmt.Println("bad")
		return Config{}
	}
	var configBby Config
	json.Unmarshal(file, &configBby)
	return configBby
}

func SetUser(user string, config Config) {
	path := getHomePath()
	config.CurrentUserName = user
	jsonData, err := json.Marshal(config)
	if err != nil {
		fmt.Println("bad")
		fmt.Println(err)
		return
	}
	err = os.WriteFile(path, jsonData, 0644)
}

func getHomePath() string {
	homePath, err := os.UserHomeDir()
	if err != nil {
		fmt.Println("bad")
		fmt.Println(err)
		return ""
	}
	return homePath + `/` + configFileName
}

func HandlerLogin(s *State, cmd Command) error {
	if len(cmd.Arguments) == 0 {
		return errors.New("need username")
	}
	_, err := s.Db.GetUser(context.Background(), cmd.Arguments[0])
	if err != nil {
		return err
	}
	SetUser(cmd.Arguments[0], *s.Conf)
	fmt.Println("User has been set")
	return nil
}

func HandlerRegister(s *State, cmd Command) error {
	if len(cmd.Arguments) == 0 {
		return errors.New("need name for user")
	}

	user, err := s.Db.CreateUser(context.Background(), database.CreateUserParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Name:      cmd.Arguments[0],
	})
	if err != nil {
		return err
	}
	s.Conf.CurrentUserName = cmd.Arguments[0]
	HandlerLogin(s, cmd)
	fmt.Printf("User was created. %v", user)
	return nil
}

func HandlerReset(s *State, cmd Command) error {
	err := s.Db.DeleteTableRows(context.Background())
	if err != nil {
		fmt.Println("table rows not deleted")
		os.Exit(1)
		return err
	}
	fmt.Println("table rows deleted")
	os.Exit(0)
	return nil
}

func HandlerGetUsers(s *State, cmd Command) error {
	users, err := s.Db.GetUsers(context.Background())
	if err != nil {
		return err
	}
	user, err := s.Db.GetUser(context.Background(), s.Conf.CurrentUserName)
	for _, u := range users {
		if u == user {
			fmt.Printf("%s (current)\n", u)
		} else {
			fmt.Println(u)
		}
	}
	return nil
}

func (c *Commands) Run(s *State, cmd Command) error {
	if _, ok := c.CommandFunc[cmd.Name]; ok {
		err := c.CommandFunc[cmd.Name](s, cmd)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *Commands) Register(name string, f func(*State, Command) error) {
	c.CommandFunc[name] = f
}
