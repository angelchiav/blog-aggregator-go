package commands

import (
	"context"
	"fmt"

	"github.com/angelchiav/blog-aggregator-go/internal/database"
)

func MiddlewareLoggedIn(
	handler func(s *State, cmd Command, user database.User) error,
) func(*State, Command) error {
	return func(s *State, cmd Command) error {
		if s == nil || s.Cfg == nil {
			return fmt.Errorf("internal error: missing state/config")
		}

		current := s.Cfg.CurrentUser

		if current == "" {
			return fmt.Errorf("you must be logged in to use: '%s'", cmd.Name)
		}

		u, err := s.DB.GetUserByName(context.Background(), current)
		if err != nil {
			return fmt.Errorf("failed to load current user: '%s': %w", current, err)
		}

		return handler(s, cmd, u)
	}
}
