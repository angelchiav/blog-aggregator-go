package commands

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/url"
	"strconv"
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
	if len(cmd.Args) < 1 {
		return fmt.Errorf("usage: agg <time_between_reqs>")
	}

	timeBetweenRequests, err := time.ParseDuration(cmd.Args[0])
	if err != nil {
		return fmt.Errorf("invalid duration %q: %w", cmd.Args[0], err)
	}

	fmt.Printf("Collecting feeds every %s\n", timeBetweenRequests)

	ticker := time.NewTicker(timeBetweenRequests)
	defer ticker.Stop()

	for {
		if err := scrapeFeeds(s); err != nil {
			log.Printf("error scraping feeds: %v\n", err)
		}
		<-ticker.C
	}
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

func scrapeFeeds(s *State) error {
	ctx := context.Background()

	feed, err := s.DB.GetNextFeedToFetch(ctx)
	if err != nil {
		return fmt.Errorf("could not get next feed to fetch: %w", err)
	}

	if err := s.DB.MarkFeedFetched(ctx, feed.ID); err != nil {
		return fmt.Errorf("could not mark feed as fetched: %w", err)
	}

	fmt.Printf("Fetching feed: %s (%s)\n", feed.Name, feed.Url)

	parsedFeed, err := rss.FetchFeed(ctx, feed.Url)
	if err != nil {
		return fmt.Errorf("could not fetch rss feed: %w", err)
	}

	for _, item := range parsedFeed.Channel.Items {
		publishedAt := sql.NullTime{}
		if t, err := parsePublishedTime(item.PubDate); err == nil {
			publishedAt = sql.NullTime{
				Time:  t,
				Valid: true,
			}
		} else {
			log.Printf("could not parse pubDate %q for feed: %s: %v\n", item.PubDate, feed.Url, err)
		}

		now := time.Now()

		err = s.DB.CreatePost(ctx, database.CreatePostParams{
			ID:        uuid.New(),
			CreatedAt: now,
			UpdatedAt: now,
			Title:     item.Title,
			Url:       item.Link,
			Description: sql.NullString{
				String: item.Description,
				Valid:  item.Description != "",
			},
			PublishedAt: publishedAt,
			FeedID:      feed.ID,
		})
		if err != nil {
			if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "unique constraint") {
				continue
			}
			log.Printf("error saving post (feed %s): %v\n", feed.Url, err)
		}
	}

	return nil
}

func parsePublishedTime(raw string) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, fmt.Errorf("empty pubDate")
	}

	layouts := []string{
		time.RFC1123Z,
		time.RFC1123,
		time.RFC822Z,
		time.RFC822,
		time.RFC3339,
	}

	for _, layout := range layouts {
		if t, err := time.Parse(layout, raw); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("could not parse pubDate: %q", raw)
}

func HandlerBrowse(s *State, cmd Command, user database.User) error {
	limit := 2
	if len(cmd.Args) >= 1 {
		n, err := strconv.Atoi(cmd.Args[0])
		if err != nil {
			return fmt.Errorf("invalid limit %q", err)
		}
		limit = n
	}

	ctx := context.Background()

	rows, err := s.DB.GetPostsForUser(ctx, database.GetPostsForUserParams{
		UserID: user.ID,
		Limit:  int32(limit),
	})
	if err != nil {
		return fmt.Errorf("could not get posts: %w", err)
	}

	if len(rows) == 0 {
		fmt.Println("No posts found.")
		return nil
	}

	for _, p := range rows {
		published := "unknown"
		if p.PublishedAt.Valid {
			published = p.PublishedAt.Time.Format(time.RFC1123)
		}

		fmt.Printf("Title: %s\nURL: %s\nPublished: %s\n\n", p.Title, p.Url, published)
	}
	return nil
}
