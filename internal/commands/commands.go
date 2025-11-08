package commands

import (
	"errors"
	"fmt"

	"github.com/angelchiav/blog-aggregator-go/internal/config"
)

type State struct {
	Cfg *config.Config
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

func HandlerLogin(s *State, cmd Command) error {
	if len(cmd.Args) < 1 {
		return errors.New("a username is required")
	}
	username := cmd.Args[0]
	if err := s.Cfg.SetUser(username); err != nil {
		return err
	}

	fmt.Printf("user set to %q\n", username)

	return nil
}
