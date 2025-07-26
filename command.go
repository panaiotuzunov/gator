package main

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/panaiotuzunov/gator/internal/config"
	"github.com/panaiotuzunov/gator/internal/database"
	"github.com/panaiotuzunov/gator/internal/fetch"
)

type state struct {
	db  *database.Queries
	cfg *config.Config
}

type command struct {
	name string
	args []string
}

type commands struct {
	list map[string]func(*state, command) error
}

func (c *commands) run(s *state, cmd command) error {
	function, ok := c.list[cmd.name]
	if !ok {
		return fmt.Errorf("error: function does not exist")
	}
	err := function(s, cmd)
	if err != nil {
		return err
	}
	return nil
}

func (c *commands) register(name string, f func(*state, command) error) {
	c.list[name] = f
}

func handlerLogin(s *state, cmd command) error {
	if len(cmd.args) != 1 {
		return fmt.Errorf("error: the login command accepts exactly one argument - username")
	}
	usernameStr := cmd.args[0]
	_, err := s.db.GetUserByName(context.Background(), usernameStr)
	if err == sql.ErrNoRows {
		return fmt.Errorf("error: user %s does not exist", usernameStr)
	} else if err != nil {
		return fmt.Errorf("error: database error - %v", err)
	}
	err = s.cfg.SetUser(usernameStr)
	if err != nil {
		return fmt.Errorf("error updating config: %v", err)
	}
	fmt.Printf("The user %s logged in successfully.\n", usernameStr)
	return nil
}

func handlerRegister(s *state, cmd command) error {
	if len(cmd.args) != 1 {
		return fmt.Errorf("error: the register command accepts exactly one argument - username")
	}
	usernameStr := cmd.args[0]
	_, err := s.db.GetUserByName(context.Background(), usernameStr)
	if err == sql.ErrNoRows {
		userData := database.CreateUserParams{
			ID:        uuid.New(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Name:      usernameStr,
		}
		CreatedUserData, err := s.db.CreateUser(context.Background(), userData)
		if err != nil {
			return fmt.Errorf("error creating user %s", usernameStr)
		}
		err = s.cfg.SetUser(usernameStr)
		if err != nil {
			return fmt.Errorf("error updating config: %v", err)
		}
		fmt.Printf("User %s created successfully. User parameters:\n%v\n", usernameStr, CreatedUserData)
	} else if err != nil {
		return fmt.Errorf("error: database error - %v", err)
	} else {
		return fmt.Errorf("error: User %s already exists", usernameStr)
	}
	return nil
}

func handlerReset(s *state, cmd command) error {
	err := s.db.DeleteUsers(context.Background())
	if err != nil {
		return fmt.Errorf("error: deletion of users failed - %v", err)
	}
	fmt.Println("Users deleted successfully")
	return nil
}

func handlerUsers(s *state, cmd command) error {
	users, err := s.db.GetUsers(context.Background())
	if err != nil {
		return fmt.Errorf("error: Reading users failed - %v", err)
	}
	for _, user := range users {
		if user == s.cfg.CurrentUserName {
			fmt.Printf("* %s (current)\n", user)
			continue
		}
		fmt.Printf("* %s\n", user)
	}
	return nil
}

func handlerAgg(s *state, cmd command) error {
	if len(cmd.args) != 1 {
		return fmt.Errorf("error: the aggregate command accepts exactly one argument - time between requests (1m0s)")
	}
	time_between_reqs, err := time.ParseDuration(cmd.args[0])
	if err != nil {
		return fmt.Errorf("error parsing time between requests arguments - %v", err)
	}
	fmt.Printf("Collecting feeds every %s\n", cmd.args[0])
	ticker := time.NewTicker(time_between_reqs)
	for ; ; <-ticker.C {
		if err := scrapeFeeds(s); err != nil {
			return fmt.Errorf("error scraping feeds - %v", err)
		}
	}
}

func handlerAddFeed(s *state, cmd command, user database.User) error {
	if len(cmd.args) != 2 {
		return fmt.Errorf("error: the addfeed command accepts exactly two argument - name, url")
	}
	feedParams := database.CreateFeedParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Name:      cmd.args[0],
		Url:       cmd.args[1],
		UserID:    user.ID,
	}
	feed, err := s.db.CreateFeed(context.Background(), feedParams)
	if err != nil {
		return fmt.Errorf("error creating feed - %v", err)
	}
	feedFollowParams := database.CreateFeedFollowParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		UserID:    user.ID,
		FeedID:    feed.ID,
	}
	_, err = s.db.CreateFeedFollow(context.Background(), feedFollowParams)
	if err != nil {
		return fmt.Errorf("error creating feed follow - %v", err)
	}
	fmt.Printf("%+v\n", feed)
	return nil
}

func handlerGetFeeds(s *state, cmd command) error {
	feeds, err := s.db.GetFeeds(context.Background())
	if err != nil {
		return fmt.Errorf("error getting feeds - %v", err)
	}
	for _, feed := range feeds {
		user, err := s.db.GetUser(context.Background(), feed.UserID)
		if err != nil {
			return fmt.Errorf("error getting user name - %v", err)
		}
		fmt.Printf("name - %v, url - %v, user - %v\n", feed.Name, feed.Url, user.Name)
	}
	return nil
}

func handlerFollow(s *state, cmd command, user database.User) error {
	if len(cmd.args) != 1 {
		return fmt.Errorf("error: the follow command accepts exactly one argument - url")
	}
	feedData, err := s.db.GetFeedByUrl(context.Background(), cmd.args[0])
	if err != nil {
		return fmt.Errorf("error getting feed data - %v", err)
	}
	feedFollowParams := database.CreateFeedFollowParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		UserID:    user.ID,
		FeedID:    feedData.ID,
	}
	feedFollowResult, err := s.db.CreateFeedFollow(context.Background(), feedFollowParams)
	if err != nil {
		return fmt.Errorf("error creating feed follow - %v", err)
	}
	fmt.Printf("User %v now follows %v feed\n", feedFollowResult.UserName, feedFollowResult.FeedName)
	return nil
}

func handlerFollowing(s *state, cmd command, user database.User) error {
	feedFollowsResult, err := s.db.GetFeedFollowsForUser(context.Background(), user.ID)
	if err != nil {
		return fmt.Errorf("error getting current user feed follows - %v", err)
	}
	if len(feedFollowsResult) == 0 {
		return fmt.Errorf("error: the current user doesn't follow any feeds")
	}
	for _, feedFollow := range feedFollowsResult {
		fmt.Printf("%+v\n", feedFollow)
	}
	return nil
}

func handlerUnfollow(s *state, cmd command, user database.User) error {
	if len(cmd.args) != 1 {
		return fmt.Errorf("error: the unfollow command accepts exactly one argument - url")
	}
	feed, err := s.db.GetFeedByUrl(context.Background(), cmd.args[0])
	if err != nil {
		return fmt.Errorf("error getting feed data - %v", err)
	}
	deleteFeedParams := database.DeleteFeedFollowParams{
		UserID: user.ID,
		FeedID: feed.ID,
	}
	if err := s.db.DeleteFeedFollow(context.Background(), deleteFeedParams); err != nil {
		return fmt.Errorf("error unfolowing feed - %v", err)
	}
	fmt.Println("Feed unfollowed successfully.")
	return nil
}

func handlerBrowse(s *state, cmd command, user database.User) error {
	postLimit := int32(2)
	if len(cmd.args) > 0 {
		limit, err := strconv.Atoi(cmd.args[0])
		if err != nil {
			return fmt.Errorf("error parsing posts limit - %v", err)
		}
		postLimit = int32(limit)
	}
	getPostsParams := database.GetPostsForUserParams{
		UserID: user.ID,
		Limit:  postLimit,
	}
	posts, err := s.db.GetPostsForUser(context.Background(), getPostsParams)
	if err != nil {
		return fmt.Errorf("error getting posts - %v", err)
	}
	if len(posts) == 0 {
		fmt.Println("There are no posts to display.")
	}
	for i, post := range posts {
		i++
		fmt.Printf("=== Post %d ===\n", i)
		fmt.Printf("Title: %s\n", post.Title)
		fmt.Printf("URL: %s\n", post.Url)
		fmt.Printf("Description: %s\n", post.Description)
		fmt.Printf("Published: %v\n", post.PublishedAt.Format("02/01/2006"))
		fmt.Println()
	}
	return nil
}

func middlewareLoggedIn(handler func(s *state, cmd command, user database.User) error) func(*state, command) error {
	return func(s *state, cmd command) error {
		if s.cfg.CurrentUserName == "" {
			return fmt.Errorf("no user is currently logged in")
		}
		currentUserStruct, err := s.db.GetUserByName(context.Background(), s.cfg.CurrentUserName)
		if err != nil {
			return fmt.Errorf("error reading user by name from DB - %v", err)
		}
		return handler(s, cmd, currentUserStruct)
	}
}

func scrapeFeeds(s *state) error {
	nextFeed, err := s.db.GetNextFeedToFetch(context.Background())
	if err != nil {
		return fmt.Errorf("error getting next feed to fetch - %v", err)
	}
	markFeedParams := database.MarkFeedFetchedParams{
		LastFetchedAt: sql.NullTime{Time: time.Now(), Valid: true},
		ID:            nextFeed.ID,
	}
	s.db.MarkFeedFetched(context.Background(), markFeedParams)
	feed, err := fetch.FetchFeed(context.Background(), nextFeed.Url)
	if err != nil {
		return err
	}
	for _, item := range feed.Channel.Item {
		parsedTime, err := parsePublishDate(item.PubDate)
		if err != nil {
			return fmt.Errorf("error parsing date: %v", err)
		}
		postParams := database.CreatePostParams{
			ID:          uuid.New(),
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
			Title:       item.Title,
			Url:         item.Link,
			Description: item.Description,
			PublishedAt: parsedTime,
			FeedID:      nextFeed.ID,
		}
		_, err = s.db.CreatePost(context.Background(), postParams)
		if err != nil {
			if strings.Contains(err.Error(), "UNIQUE constraint failed") ||
				strings.Contains(err.Error(), "duplicate key") {
				fmt.Printf("Post %s already exists. Skipping... \n", item.Title)
				continue
			}
			fmt.Printf("Creating post %s failed with error - %v. Skipping...\n", item.Title, err)
		}

	}
	return nil
}

func parsePublishDate(dateStr string) (time.Time, error) {
	formats := []string{
		time.RFC822Z,                     // "Mon, 02 Jan 2006 15:04:05 -0700" (numeric timezone)
		time.RFC822,                      // "Mon, 02 Jan 2006 15:04:05 MST" (timezone abbreviation)
		time.RFC3339,                     // "2006-01-02T15:04:05Z07:00"
		"Mon, 2 Jan 2006 15:04:05 -0700", // Single digit day with numeric timezone
	}

	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse date: %s", dateStr)
}
