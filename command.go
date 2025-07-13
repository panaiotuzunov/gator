package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/panaiotuzunov/gator/internal/config"
	"github.com/panaiotuzunov/gator/internal/database"
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
		return fmt.Errorf("%v", err)
	}
	return nil
}

func (c *commands) register(name string, f func(*state, command) error) {
	c.list[name] = f
}

func handlerLogin(s *state, cmd command) error {
	if len(cmd.args) != 1 {
		return fmt.Errorf("error: the login handler accepts exactly one argument - username")
	}
	usernameStr := cmd.args[0]
	_, err := s.db.GetUserByName(context.Background(), sql.NullString{String: usernameStr, Valid: true})
	if err == sql.ErrNoRows {
		fmt.Printf("User %s does not exist.\n", usernameStr)
		os.Exit(1)
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
		return fmt.Errorf("error: the register handler accepts exactly one argument - username")
	}
	usernameStr := cmd.args[0]
	_, err := s.db.GetUserByName(context.Background(), sql.NullString{String: usernameStr, Valid: true})
	if err == sql.ErrNoRows {
		userData := database.CreateUserParams{
			ID:        uuid.New(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Name:      sql.NullString{String: usernameStr, Valid: true},
		}
		CreatedUserData, err := s.db.CreateUser(context.Background(), userData)
		if err != nil {
			fmt.Printf("Error creating user %s.\n", usernameStr)
			os.Exit(1)
		}
		err = s.cfg.SetUser(usernameStr)
		if err != nil {
			return fmt.Errorf("error updating config: %v", err)
		}
		fmt.Printf("User %s created successfully. User parameters:\n%v", usernameStr, CreatedUserData)
	} else if err != nil {
		return fmt.Errorf("error: database error - %v", err)
	} else {
		fmt.Printf("User %s already exists.\n", usernameStr)
		os.Exit(1)
	}
	return nil
}
