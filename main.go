package main

import (
	"fmt"
	"log"
	"os"

	"github.com/felipemarinho97/dev-spaces/cli"
	"github.com/felipemarinho97/dev-spaces/config"
)

func main() {
	err := config.LoadConfig()
	if err != nil {
		log.Fatal(fmt.Errorf("error loading config: %s\nPlease create a config.toml file or see: https://github.com/felipemarinho97/dev-spaces/blob/master/CONFIGURATION.md", err))
	}

	app := cli.GetCLI()

	err = app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
