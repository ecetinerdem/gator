package config

import (
	"context"
	"database/sql"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
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

func fetchFeed(ctx context.Context, feedURL string) (*database.RSSFeed, error) {
	request, err := http.NewRequestWithContext(ctx, "GET", feedURL, nil)

	if err != nil {
		return nil, err
	}

	request.Header.Set("User-Agent", "gator")

	client := &http.Client{}
	resp, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)

	if err != nil {
		return nil, err
	}

	var feed database.RSSFeed

	err = xml.Unmarshal(data, &feed)

	if err != nil {
		return nil, err
	}

	feed.Channel.Title = html.UnescapeString(feed.Channel.Title)
	feed.Channel.Description = html.UnescapeString(feed.Channel.Description)

	for i := range feed.Channel.Item {
		feed.Channel.Item[i].Title = html.UnescapeString(feed.Channel.Item[i].Title)
		feed.Channel.Item[i].Description = html.UnescapeString(feed.Channel.Item[i].Description)
	}

	return &feed, nil
}

func AddFeedHandler(s *State, cmd Command, user database.User) error {
	if len(cmd.Args) < 2 {
		return errors.New("usage: addfeed <name> <url>")
	}

	name := cmd.Args[0]
	url := cmd.Args[1]

	// Create the feed (no need to get user anymore!)
	feed, err := s.DB.CreateFeed(context.Background(), database.CreateFeedParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Name:      name,
		Url:       url,
		UserID:    user.ID,
	})
	if err != nil {
		return fmt.Errorf("failed to create feed: %w", err)
	}

	// Automatically create a feed follow for the user
	feedFollow, err := s.DB.CreateFeedFollow(context.Background(), database.CreateFeedFollowParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		UserID:    user.ID,
		FeedID:    feed.ID,
	})
	if err != nil {
		return fmt.Errorf("failed to create feed follow: %w", err)
	}

	fmt.Printf("Feed created successfully:\n")
	fmt.Printf("* Name: %s\n", feed.Name)
	fmt.Printf("* URL: %s\n", feed.Url)
	fmt.Printf("* User: %s\n", user.Name)
	fmt.Printf("\n%s is now following %s\n", feedFollow.UserName, feedFollow.FeedName)

	return nil
}

func FeedsHandler(s *State, cmd Command) error {

	feeds, err := s.DB.GetFeeds(context.Background())

	if err != nil {
		return fmt.Errorf("failed to get get feeds: %w", err)
	}

	if len(feeds) == 0 {
		fmt.Println("No feeds found")
		return nil
	}

	for _, feed := range feeds {
		user, err := s.DB.GetUserByID(context.Background(), feed.UserID)

		if err != nil {
			return fmt.Errorf("failed to get user for feed %s: %w", feed.Name, err)
		}

		fmt.Printf("* Name: %s\n", feed.Name)
		fmt.Printf("  URL: %s\n", feed.Url)
		fmt.Printf("  User: %s\n", user.Name)
		fmt.Println()

	}
	return nil

}

func FollowHandler(s *State, cmd Command, user database.User) error {
	if len(cmd.Args) < 1 {
		return errors.New("usage: follow <url>")
	}

	url := cmd.Args[0]

	// Get feed by URL (no need to get user anymore!)
	feed, err := s.DB.GetFeedByURL(context.Background(), url)
	if err != nil {
		return fmt.Errorf("failed to get feed: %w", err)
	}

	// Create feed follow
	feedFollow, err := s.DB.CreateFeedFollow(context.Background(), database.CreateFeedFollowParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		UserID:    user.ID,
		FeedID:    feed.ID,
	})
	if err != nil {
		return fmt.Errorf("failed to create feed follow: %w", err)
	}

	fmt.Printf("%s is now following %s\n", feedFollow.UserName, feedFollow.FeedName)

	return nil
}

func UnfollowHandler(s *State, cmd Command, user database.User) error {
	if len(cmd.Args) < 1 {
		return errors.New("usage: unfollow <url>")
	}

	url := cmd.Args[0]

	// Get feed by URL
	feed, err := s.DB.GetFeedByURL(context.Background(), url)
	if err != nil {
		return fmt.Errorf("failed to get feed: %w", err)
	}

	// Delete feed follow
	err = s.DB.DeleteFeedFollow(context.Background(), database.DeleteFeedFollowParams{
		UserID: user.ID,
		FeedID: feed.ID,
	})
	if err != nil {
		return fmt.Errorf("failed to unfollow feed: %w", err)
	}

	fmt.Printf("%s has unfollowed %s\n", user.Name, feed.Name)

	return nil
}

func FollowingHandler(s *State, cmd Command, user database.User) error {
	// Get all feed follows for user (no need to get user anymore!)
	feedFollows, err := s.DB.GetFeedFollowsForUser(context.Background(), user.ID)
	if err != nil {
		return fmt.Errorf("failed to get feed follows: %w", err)
	}

	if len(feedFollows) == 0 {
		fmt.Println("Not following any feeds")
		return nil
	}

	fmt.Printf("Feeds %s is following:\n", user.Name)
	for _, ff := range feedFollows {
		fmt.Printf("* %s\n", ff.FeedName)
	}

	return nil
}

func AggHandler(s *State, cmd Command) error {
	if len(cmd.Args) < 1 {
		return errors.New("usage: agg <time_between_reqs>")
	}

	timeBetweenRequests, err := time.ParseDuration(cmd.Args[0])
	if err != nil {
		return fmt.Errorf("invalid duration: %w", err)
	}

	fmt.Printf("Collecting feeds every %s\n", timeBetweenRequests)

	ticker := time.NewTicker(timeBetweenRequests)
	defer ticker.Stop()

	// Run immediately, then on each tick
	for ; ; <-ticker.C {
		err := scrapeFeeds(s)
		if err != nil {
			fmt.Printf("Error scraping feeds: %v\n", err)
		}
	}
}

func scrapeFeeds(s *State) error {
	// Get the next feed to fetch
	feed, err := s.DB.GetNextFeedToFetch(context.Background())
	if err != nil {
		return fmt.Errorf("couldn't get next feed to fetch: %w", err)
	}

	// Mark it as fetched
	err = s.DB.MarkFeedFetched(context.Background(), feed.ID)
	if err != nil {
		return fmt.Errorf("couldn't mark feed as fetched: %w", err)
	}

	// Fetch the feed
	rssFeed, err := fetchFeed(context.Background(), feed.Url)
	if err != nil {
		return fmt.Errorf("couldn't fetch feed %s: %w", feed.Name, err)
	}

	fmt.Printf("Feed %s collected, %d posts found\n", feed.Name, len(rssFeed.Channel.Item))

	// Save each post to the database
	for _, item := range rssFeed.Channel.Item {
		// Parse the published date
		publishedAt, err := parseDate(item.PubDate)
		if err != nil {
			fmt.Printf("couldn't parse date %s: %v\n", item.PubDate, err)
			continue
		}

		// Create the post
		_, err = s.DB.CreatePost(context.Background(), database.CreatePostParams{
			ID:        uuid.New(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
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
			// Check if it's a duplicate URL error
			if strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
				continue // Skip duplicate posts
			}
			fmt.Printf("couldn't create post: %v\n", err)
			continue
		}
	}

	return nil
}

func parseDate(dateStr string) (time.Time, error) {
	// Common RSS date formats
	formats := []string{
		time.RFC1123Z,
		time.RFC1123,
		time.RFC822Z,
		time.RFC822,
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02 15:04:05",
		"Mon, 02 Jan 2006 15:04:05 -0700",
	}

	for _, format := range formats {
		t, err := time.Parse(format, dateStr)
		if err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("couldn't parse date: %s", dateStr)
}

func BrowseHandler(s *State, cmd Command, user database.User) error {
	limit := 2 // Default limit

	if len(cmd.Args) > 0 {
		parsedLimit, err := strconv.Atoi(cmd.Args[0])
		if err != nil {
			return fmt.Errorf("invalid limit: %w", err)
		}
		limit = parsedLimit
	}

	posts, err := s.DB.GetPostsForUser(context.Background(), database.GetPostsForUserParams{
		UserID: user.ID,
		Limit:  int32(limit),
	})
	if err != nil {
		return fmt.Errorf("couldn't get posts: %w", err)
	}

	if len(posts) == 0 {
		fmt.Println("No posts found. Follow some feeds first!")
		return nil
	}

	fmt.Printf("Found %d posts for user %s:\n", len(posts), user.Name)
	fmt.Println()

	for _, post := range posts {
		fmt.Printf("Title: %s\n", post.Title)
		fmt.Printf("URL: %s\n", post.Url)
		if post.Description.Valid {
			// Truncate description if too long
			desc := post.Description.String
			if len(desc) > 200 {
				desc = desc[:200] + "..."
			}
			fmt.Printf("Description: %s\n", desc)
		}
		fmt.Printf("Published: %s\n", post.PublishedAt.Format("2006-01-02 15:04:05"))
		fmt.Println("=====================================")
	}

	return nil
}
