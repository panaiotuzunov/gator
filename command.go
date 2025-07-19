package main

import (
	"context"
	"database/sql"
	"fmt"
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
	feed, err := fetch.FetchFeed(context.Background(), "https://www.wagslane.dev/index.xml")
	if err != nil {
		return fmt.Errorf("error fetching feed - %v", err)
	}
	fmt.Println(feed)
	return nil
}

func handlerAddFeed(s *state, cmd command) error {
	if len(cmd.args) != 2 {
		return fmt.Errorf("error: the addfeed command accepts exactly two argument - name, url")
	}
	currentUserStruct, err := s.db.GetUserByName(context.Background(), s.cfg.CurrentUserName)
	if err != nil {
		return fmt.Errorf("error reading user by name from DB - %v", err)
	}
	feedParams := database.CreateFeedParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Name:      cmd.args[0],
		Url:       cmd.args[1],
		UserID:    currentUserStruct.ID,
	}
	feed, err := s.db.CreateFeed(context.Background(), feedParams)
	if err != nil {
		return fmt.Errorf("error creating feed - %v", err)
	}
	feedFollowParams := database.CreateFeedFollowParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		UserID:    currentUserStruct.ID,
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

func handlerFollow(s *state, cmd command) error {
	if len(cmd.args) != 1 {
		return fmt.Errorf("error: the follow command accepts exactly one argument - url")
	}
	currentUser, err := s.db.GetUserByName(context.Background(), s.cfg.CurrentUserName)
	if err != nil {
		return fmt.Errorf("error getting current user data - %v", err)
	}
	feedData, err := s.db.GetFeedByUrl(context.Background(), cmd.args[0])
	if err != nil {
		return fmt.Errorf("error getting feed data - %v", err)
	}
	feedFollowParams := database.CreateFeedFollowParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		UserID:    currentUser.ID,
		FeedID:    feedData.ID,
	}
	feedFollowResult, err := s.db.CreateFeedFollow(context.Background(), feedFollowParams)
	if err != nil {
		return fmt.Errorf("error creating feed follow - %v", err)
	}
	fmt.Printf("User %v now follows %v feed\n", feedFollowResult.UserName, feedFollowResult.FeedName)
	return nil
}

func handlerFollowing(s *state, cmd command) error {
	userData, err := s.db.GetUserByName(context.Background(), s.cfg.CurrentUserName)
	if err != nil {
		return fmt.Errorf("error getting current user data - %v", err)
	}
	feedFollowsResult, err := s.db.GetFeedFollowsForUser(context.Background(), userData.ID)
	if err != nil {
		return fmt.Errorf("error getting current user feed follows - %v", err)
	}
	for _, feedFollow := range feedFollowsResult {
		fmt.Printf("%+v\n", feedFollow)
	}
	return nil
}
