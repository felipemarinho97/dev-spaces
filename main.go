package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/felipemarinho97/dev-spaces/handlers"
	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name: "dev-spaces",
		Authors: []*cli.Author{
			{
				Name:  "Felipe Marinho",
				Email: "felipevm97@gmail.com",
			},
		},

		Usage: "CLI to help dev-spaces creation and management",
		Action: func(c *cli.Context) error {
			fmt.Println("please see --help")

			return nil
		},
	}
	app.EnableBashCompletion = true

	app.Commands = []*cli.Command{
		{
			Name:        "start",
			Description: "Starts the dev environment by placing a spot request",
			Action:      handlers.Create,
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
				&cli.DurationFlag{
					Name:    "timeout",
					Aliases: []string{"t"},
					Value:   time.Hour * 1,
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
		{
			Name:   "status",
			Action: handlers.Status,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:    "name",
					Aliases: []string{"n"},
					Value:   "",
				},
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
