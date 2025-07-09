package main

import (
	"fmt"

	"github.com/panaiotuzunov/gator/internal/config"
)

func main() {
	configStruct, err := config.Read()
	if err != nil {
		fmt.Println("oouuppsss")
	}
	err = configStruct.SetUser("Gosho")
	if err != nil {
		fmt.Println("oouuppsss")
	} else {
		fmt.Println("Success!")
	}
}
