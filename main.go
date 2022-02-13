package main

import (
	"fmt"
	"log"
	"os"

	"github.com/felipemarinho97/dev-spaces/handlers"
	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:  "dev-space",
		Usage: "create a dev-space for you",
		Action: func(c *cli.Context) error {
			fmt.Println("please see --help")

			return nil
		},

		Commands: []*cli.Command{
			{
				Name:   "start",
				Action: handlers.Create,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "name",
						Aliases: []string{"n"},
						Value:   "",
					},
					&cli.IntFlag{
						Name:    "min-cpus",
						Aliases: []string{"c"},
						Value:   1,
					},
					&cli.IntFlag{
						Name:    "min-memory",
						Aliases: []string{"m"},
						Value:   1,
					},
					&cli.StringFlag{
						Name:  "max-price",
						Value: "0.08",
					},
				},
			},
			{
				Name:   "stop",
				Action: handlers.Stop,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "name",
						Aliases: []string{"n"},
						Value:   "",
					},
				},
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
