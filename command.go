package main

import (
	"fmt"

	"github.com/panaiotuzunov/gator/internal/config"
)

type state struct {
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
	err := s.cfg.SetUser(cmd.args[0])
	if err != nil {
		return err
	}
	fmt.Println("The user was set successfully.")
	return nil
}
