package main

import (
	"fmt"
	"os"

	"github.com/panaiotuzunov/gator/internal/config"
)

func main() {
	configStruct, err := config.Read()
	if err != nil {
		fmt.Printf("cound not read config file - %v\n", err)
		os.Exit(1)
	}
	stateStruct := state{cfg: &configStruct}
	cmds := &commands{
		list: make(map[string]func(*state, command) error),
	}
	cmds.register("login", handlerLogin)
	if len(os.Args) < 2 {
		fmt.Println("error: not enough arguments")
		os.Exit(1)
	}
	cmd := command{name: os.Args[1], args: os.Args[2:]}
	err = cmds.run(&stateStruct, cmd)
	if err != nil {
		fmt.Printf("running command %v failed with %v\n", cmd.name, err)
		os.Exit(1)
	}
}
