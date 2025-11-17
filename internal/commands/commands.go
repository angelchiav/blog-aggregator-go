package commands

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"strings"
	"time"

	"github.com/angelchiav/blog-aggregator-go/internal/config"
	"github.com/angelchiav/blog-aggregator-go/internal/database"
	"github.com/angelchiav/blog-aggregator-go/internal/rss"
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

func (s *State) HandlerAgg(cmd Command) error {
	feed, err := rss.FetchFeed(context.Background(), "https://www.wagslane.dev/index.xml")
	if err != nil {
		return err
	}
	fmt.Printf("%+v\n", feed)
	return nil
}

func HandlerAddFeed(s *State, cmd Command, user database.User) error {
	if len(cmd.Args) < 2 {
		return fmt.Errorf("usage: addfeed <name> <url>")
	}

	name := strings.TrimSpace(cmd.Args[0])
	feedURL := strings.TrimSpace(cmd.Args[1])

	if _, err := url.ParseRequestURI(feedURL); err != nil {
		return fmt.Errorf("invalid url: %v", err)
	}

	feed, err := s.addFeed(user, name, feedURL)
	if err != nil {
		return err
	}

	ctx := context.Background()

	_, err = s.DB.CreateFeedFollow(ctx, database.CreateFeedFollowParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		UserID:    user.ID,
		FeedID:    feed.ID,
	})
	if err != nil && !strings.Contains(err.Error(), "duplicate key") {
		return fmt.Errorf("create follow: %w", err)
	}

	fmt.Printf("feed created | %s (%s)\n", feed.Name, feed.Url)

	return nil
}

func (s *State) HandlerGetFeed(cmd Command) error {

	feed, err := s.DB.GetFeed(context.Background())
	if err != nil {
		return fmt.Errorf("no feed found: %v", err)
	}

	for _, f := range feed {

		id, err := s.DB.GetUserNameById(context.Background(), f.UserID)
		if err != nil {
			return fmt.Errorf("no user with this id: %v", err)
		}

		fmt.Printf("%s (%s) - (%s)\n", f.Name, f.Url, id)
	}
	return nil
}

func (s *State) addFeed(user database.User, name string, url string) (database.Feed, error) {
	feed, err := s.DB.CreateFeed(context.Background(), database.CreateFeedParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Name:      name,
		Url:       url,
		UserID:    user.ID,
	})
	if err != nil {
		return database.Feed{}, fmt.Errorf("error creating feed: %v", err)
	}

	return feed, nil
}

func feedFollow(s *State, user database.User, url string) (database.CreateFeedFollowRow, error) {
	feed, err := s.DB.GetFeedByURL(context.Background(), url)
	if err != nil {
		return database.CreateFeedFollowRow{}, fmt.Errorf("feed url does not exist: %v", err)
	}

	feedFollow, err := s.DB.CreateFeedFollow(context.Background(), database.CreateFeedFollowParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		UserID:    user.ID,
		FeedID:    feed.ID,
	})
	if err != nil {
		return database.CreateFeedFollowRow{}, fmt.Errorf("feed follow cannot be created: %v", err)
	}

	return feedFollow, nil
}

func HandlerFeedFollow(s *State, cmd Command, user database.User) error {
	if len(cmd.Args) < 1 {
		return fmt.Errorf("usage: follow <url>")
	}

	url := cmd.Args[0]

	feedfollow, err := feedFollow(s, user, url)
	if err != nil {
		return err
	}

	fmt.Printf("%v - (%v)\n", feedfollow.FeedName, feedfollow.UserName)

	return nil
}

func HandlerFeedFollowing(s *State, cmd Command, user database.User) error {
	ctx := context.Background()

	rows, err := s.DB.GetFeedFollowsForUser(ctx, user.ID)
	if err != nil {
		return fmt.Errorf("could not fetch follows: %v", err)
	}

	if len(rows) == 0 {
		fmt.Println("No followed feeds.")
		return nil
	}

	for _, row := range rows {
		fmt.Printf("- %s (%s)\n", row.FeedName, row.UserName)
	}

	return nil
}

func feedUnfollow(s *State, user database.User, url string) error {
	ctx := context.Background()

	feed, err := s.DB.GetFeedByURL(ctx, url)
	if err != nil {
		return fmt.Errorf("feed url does not exist: %v", err)
	}

	err = s.DB.DeleteFeedFollowRecord(ctx, database.DeleteFeedFollowRecordParams{
		UserID: user.ID,
		FeedID: feed.ID,
	})
	if err != nil {
		return fmt.Errorf("could not unfollow feed: %v", err)
	}

	return nil
}

func HandlerFeedUnfollow(s *State, cmd Command, user database.User) error {
	if len(cmd.Args) < 1 {
		return fmt.Errorf("usage: unfollow <url>")
	}

	url := cmd.Args[0]

	if err := feedUnfollow(s, user, url); err != nil {
		return err
	}

	fmt.Printf("unfollowed feed: %s (%s)\n", url, user.Name)

	return nil
}
