package commands

import (
	"context"
	"fmt"
	"strings"

	"github.com/angelchiav/blog-aggregator-go/internal/database"
)

func (s *State) HandlerAddFeed(cmd Command, user database.User) error {
	var name string
	var feedURL string

	if len(cmd.Args) > 0 {
		feedURL = cmd.Args[0]
	}

	if len(cmd.Args) > 1 {
		name = strings.Join(cmd.Args[:1], " ")
	}

	if strings.TrimSpace(feedURL) == "" {
		return fmt.Errorf("url feed missing")
	}

	feed, err := s.addFeed(user, name, feedURL)
	if err != nil {
		return err
	}

	ctx := context.Background()

	_, err = s.DB.CreateFeedFollow(ctx, database.CreateFeedFollowParams{})
}

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
