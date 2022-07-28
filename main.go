package main

import (
	"log"
	"os"

	"github.com/felipemarinho97/dev-spaces/cli"
	"github.com/felipemarinho97/dev-spaces/config"
)

func main() {
	err := config.LoadConfig()
	if err != nil {
		log.Fatal(err)
	}

	app := cli.GetCLI()

	err = app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
