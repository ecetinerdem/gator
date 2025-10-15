package config

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/ecetinerdem/gator/internal/database"
	"github.com/google/uuid"
)

const configFilename = ".gatorconfig.json"

type Config struct {
	DBURL           string `json:"db_url"`
	CurrentUserName string `json:"current_user_name"`
}

func (cfg *Config) SetUser(userName string) error {
	cfg.CurrentUserName = userName
	return write(*cfg)
}

func Read() (Config, error) {
	fullPath, err := getConfigFilePath()
	if err != nil {
		return Config{}, err
	}

	file, err := os.Open(fullPath)
	if err != nil {
		return Config{}, err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	cfg := Config{}
	err = decoder.Decode(&cfg)
	if err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func getConfigFilePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	fullPath := filepath.Join(home, configFilename)
	return fullPath, nil
}

func write(cfg Config) error {

	fullPath, err := getConfigFilePath()
	if err != nil {
		return err
	}

	file, err := os.Create(fullPath)
	if err != nil {
		return err
	}

	defer file.Close()

	encoder := json.NewEncoder(file)
	err = encoder.Encode(cfg)

	if err != nil {
		return err
	}
	return nil
}

type State struct {
	Cfg *Config
	DB  *database.Queries
}

type Command struct {
	Name string
	Args []string
}

func LoginHandler(s *State, cmd Command) error {

	if len(cmd.Args) < 1 {
		return errors.New("missing username")
	}

	username := cmd.Args[0]

	// verify user exists
	_, err := s.DB.GetUser(context.Background(), username)
	if err != nil {
		// if not found, return error so exit code is 1
		return fmt.Errorf("user %q does not exist: %w", username, err)
	}

	if err := s.Cfg.SetUser(username); err != nil {
		return err
	}

	fmt.Println("User has been set")

	return nil
}

func RegisterHandler(s *State, cmd Command) error {

	if len(cmd.Args) < 1 {
		return errors.New("missing username")
	}

	username := cmd.Args[0]

	ctx := context.Background()
	if _, err := s.DB.CreateUser(ctx, database.CreateUserParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Name:      username,
	}); err != nil {
		return err
	}

	if err := s.Cfg.SetUser(username); err != nil {
		return err
	}

	fmt.Println("User has been created")

	return nil
}

func ResetHandler(s *State, cmd Command) error {
	err := s.DB.DeleteUsers(context.Background())

	if err != nil {
		return fmt.Errorf("failed to delete users: %w", err)
	}

	fmt.Println("All users have been deleted")
	return nil
}

func GetUsersHandler(s *State, cmd Command) error {

	users, err := s.DB.GetUsers(context.Background())
	if err != nil {
		return fmt.Errorf("failed to get users: %w", err)
	}

	for _, user := range users {
		if user.Name == s.Cfg.CurrentUserName {
			fmt.Printf("* %s (current)\n", user.Name)
		} else {
			fmt.Printf("* %s\n", user.Name)
		}
	}

	return nil
}

type Commands struct {
	MapC map[string]func(s *State, c Command) error
}

func (c *Commands) Run(s *State, cmd Command) error {

	handler, exists := c.MapC[cmd.Name]
	if !exists {
		return fmt.Errorf("unknown command: %s", cmd.Name)
	}

	return handler(s, cmd)
}

func (c *Commands) Register(Name string, f func(*State, Command) error) {
	if c.MapC == nil {
		c.MapC = make(map[string]func(s *State, c Command) error)
	}

	c.MapC[Name] = f
}
