package main

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/lib/pq"
	"github.com/panaiotuzunov/gator/internal/config"
	"github.com/panaiotuzunov/gator/internal/database"
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
	db, err := sql.Open("postgres", configStruct.DbUrl)
	if err != nil {
		fmt.Printf("Could connect to SQL DB - %v\n", err)
	}
	stateStruct.db = database.New(db)
	cmds.register("login", handlerLogin)
	if len(os.Args) < 2 {
		fmt.Println("error: not enough arguments")
		os.Exit(1)
	}
	cmds.register("register", handlerRegister)
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
