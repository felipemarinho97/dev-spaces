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
					Name:     "name",
					Required: true,
					Aliases:  []string{"n"},
					Value:    "",
				},
				&cli.IntFlag{
					Name:    "min-cpus",
					Aliases: []string{"c"},
					Value:   1,
				},
				&cli.Float64Flag{
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
				&cli.StringFlag{
					Name:  "region",
					Value: os.Getenv("AWS_REGION"),
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
				&cli.StringFlag{
					Name:  "region",
					Value: os.Getenv("AWS_REGION"),
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
				&cli.StringFlag{
					Name:  "region",
					Value: os.Getenv("AWS_REGION"),
				},
			},
		},
		{
			Name:        "bootstrap",
			Description: "Create a the dev space environment",
			Action:      handlers.Bootstrap,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:     "name",
					Aliases:  []string{"n"},
					Required: false,
					Value:    "",
				},
				&cli.StringFlag{
					Name:     "template",
					Aliases:  []string{"t"},
					Value:    "",
					Required: true,
				},
				&cli.StringFlag{
					Name:  "region",
					Value: os.Getenv("AWS_REGION"),
				},
			},
		},
		{
			Name:        "destroy",
			Description: "Destroy a dev space",
			Action:      handlers.Destroy,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:    "name",
					Aliases: []string{"n"},
					Value:   "",
				},
				&cli.StringFlag{
					Name:  "region",
					Value: os.Getenv("AWS_REGION"),
				},
			},
		},
		{
			Name:        "list",
			Description: "List all the dev spaces",
			Action:      handlers.ListTemplates,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:  "region",
					Value: os.Getenv("AWS_REGION"),
				},
				&cli.StringFlag{
					Name:    "output",
					Usage:   "Output format: short or wide",
					Aliases: []string{"o"},
					Value:   "short",
				},
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
