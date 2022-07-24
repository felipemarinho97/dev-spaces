package main

import (
	"log"
	"os"

	"github.com/felipemarinho97/dev-spaces/cli"
)

func main() {
	app := cli.GetCLI()

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
