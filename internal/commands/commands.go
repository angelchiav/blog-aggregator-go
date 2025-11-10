package commands

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/angelchiav/blog-aggregator-go/internal/config"
	"github.com/angelchiav/blog-aggregator-go/internal/database"
	"github.com/google/uuid"
)

type State struct {
	Cfg *config.Config
	DB  *database.Queries
}

type Command struct {
	Name string
	Args []string
}

type Commands struct {
	Handlers map[string]func(*State, Command) error
}

func (c *Commands) Register(name string, f func(*State, Command) error) {
	if c.Handlers == nil {
		c.Handlers = make(map[string]func(*State, Command) error)
	}
	c.Handlers[name] = f
}

func (c *Commands) Run(s *State, cmd Command) error {
	h, ok := c.Handlers[cmd.Name]
	if !ok {
		return fmt.Errorf("unknown command: %s", cmd.Name)
	}
	return h(s, cmd)
}

func (s *State) HandlerLogin(cmd Command) error {
	if len(cmd.Args) < 1 {
		return fmt.Errorf("a username is required")
	}
	username := strings.TrimSpace(cmd.Args[0])

	// Verify user exists in DB
	if _, err := s.DB.GetUserByName(context.Background(), username); err != nil {
		return fmt.Errorf("no such user")
	}

	if err := s.Cfg.SetUser(username); err != nil {
		return err
	}

	fmt.Printf("user set to %q\n", username)
	return nil
}

func (s *State) HandlerRegister(cmd Command) error {
	if len(cmd.Args) < 1 {
		return fmt.Errorf("missing <name>")
	}
	name := strings.TrimSpace(cmd.Args[0])
	now := time.Now()

	user, err := s.DB.CreateUser(context.Background(), database.CreateUserParams{
		ID:        uuid.New(),
		CreatedAt: now,
		UpdatedAt: now,
		Name:      name,
	})
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			return fmt.Errorf("user already exists")
		}
		return fmt.Errorf("CreateUser: %w", err)
	}

	if err := s.Cfg.SetUser(name); err != nil {
		return fmt.Errorf("set user in config: %w", err)
	}

	fmt.Printf("user created: %s\n", user.Name)
	log.Printf("DEBUG USER: ID=%s CreatedAt=%s UpdatedAt=%s Name=%s",
		user.ID, user.CreatedAt.Format(time.RFC3339), user.UpdatedAt.Format(time.RFC3339), user.Name)

	return nil
}

func (s *State) HandlerReset(cmd Command) error {
	if err := s.DB.Reset(context.Background()); err != nil {
		return err
	}
	fmt.Println("All users deleted.")
	return nil
}

func (s *State) HandlerUsers(cmd Command) error {
	users, err := s.DB.GetUsers(context.Background())
	if err != nil {
		return fmt.Errorf("error fetching users: %v", err)
	}

	current := strings.TrimSpace(s.Cfg.CurrentUser)

	if len(users) == 0 {
		fmt.Println("No users available.")
		return nil
	}

	for _, user := range users {
		if user.Name == current {
			fmt.Printf("* %s (current)\n", user.Name)
		} else {
			fmt.Printf("* %s\n", user.Name)
		}
	}
	return nil
}
