package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/felipemarinho97/dev-spaces/handlers"
	v2 "github.com/felipemarinho97/dev-spaces/handlers/v2"
	"github.com/urfave/cli/v2"
)

const (
	ADM       = "ADMINISTRATION"
	LIFECYCLE = "DEV-SPACE"
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
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "region",
				Aliases: []string{"r"},
				Value:   "us-east-1",
				Usage:   "AWS region",
				EnvVars: []string{"AWS_REGION"},
			},
		},
		EnableBashCompletion: true,
		Usage:                "CLI to help dev-spaces creation and management",
		Action: func(c *cli.Context) error {
			fmt.Println("please see --help")

			return nil
		},
		Compiled: time.Now(),
	}

	app.Commands = []*cli.Command{
		{
			Name:        "start",
			Description: "Starts the dev environment by placing a spot request.",
			Usage:       "-n <name> [-c <min-cpus> -m <min-memory> --max-price <max-price> -t <timeout>]",
			Category:    LIFECYCLE,
			Action:      handlers.Create,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:     "name",
					Required: true,
					Aliases:  []string{"n"},
					Usage:    "The name of the dev-space",
				},
				&cli.IntFlag{
					Name:    "min-cpus",
					Aliases: []string{"c"},
					Value:   0,
					Usage:   "Minimum number of CPUs",
				},
				&cli.Float64Flag{
					Name:    "min-memory",
					Aliases: []string{"m"},
					Value:   0,
					Usage:   "Minimum amount of memory in GB",
				},
				&cli.StringFlag{
					Name:  "max-price",
					Value: "0.50",
					Usage: "Maximum price per hour for the spot request",
				},
				&cli.DurationFlag{
					Name:    "timeout",
					Aliases: []string{"t"},
					Value:   time.Hour * 1,
					Usage:   "Timeout for the spot request",
				},
			},
		},
		{
			Name:        "stop",
			Description: "Stops the dev environment by canceling the spot request.",
			Usage:       "[-n <name>]",
			Category:    LIFECYCLE,
			Action:      handlers.Stop,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:    "name",
					Aliases: []string{"n"},
					Usage:   "The name of the dev-space",
				},
			},
		},
		{
			Name:        "status",
			Description: "Shows the status of the most recent dev-space requests.",
			Usage:       "[-n <name>]",
			Category:    LIFECYCLE,
			Action:      handlers.Status,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:    "name",
					Aliases: []string{"n"},
					Usage:   "The name of the dev-space",
				},
			},
		},
		{
			Name:        "create",
			Description: "Create a the dev space environment automatically.",

			Category: ADM,
			Action:   v2.BootstrapV2,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:     "name",
					Aliases:  []string{"n"},
					Required: true,
				},
				&cli.StringFlag{
					Name:     "key-name",
					Aliases:  []string{"k"},
					Required: true,
					Usage:    "Name of the SSH key pair to use",
				},
				&cli.StringFlag{
					Name:     "ami",
					Aliases:  []string{"i"},
					Required: true,
					Usage:    "Amazon Machine Image to use",
				},
				&cli.StringFlag{
					Name:     "instance-profile-arn",
					Aliases:  []string{"p"},
					Required: true,
					Usage:    "Instance profile ARN (arn:aws:iam::<account-id>:instance-profile/<instance-profile-name>) to use.",
				},
				&cli.IntFlag{
					Name:        "storage-size",
					Aliases:     []string{"s"},
					DefaultText: "1GB",
					Value:       1,
					Usage:       "Storage size in GB to use",
				},
				&cli.StringFlag{
					Name:    "prefered-instance-type",
					Aliases: []string{"t"},
					Value:   "t2.micro",
					Usage:   "Prefered instance type to use, this will optimize the price for this type",
				},
			},
			Usage: "-n <name> -k <key-name> -i <ami> -p <instance-profile-arn> [-s <storage-size> -t <prefered-instance-type>]",
		},
		{
			Name:        "list",
			Description: "List all the dev spaces",
			Category:    LIFECYCLE,
			Action:      handlers.ListTemplates,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:    "output",
					Usage:   "Output format: short or wide",
					Aliases: []string{"o"},
					Value:   "short",
				},
			},
			Usage: "[-o <output>]",
		},
		{
			Name:        "bootstrap",
			Description: "Bootstrap a the dev space environment from an template file (Advanced)",
			Category:    ADM,
			Action:      handlers.Bootstrap,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:    "name",
					Aliases: []string{"n"},
					Usage:   "The name of the dev-space",
				},
				&cli.StringFlag{
					Name:     "template",
					Aliases:  []string{"t"},
					Required: true,
					Usage:    "The template (file ou url) to use",
				},
				&cli.StringFlag{
					Name:    "region",
					EnvVars: []string{"AWS_REGION"},
				},
			},
			Usage: "-t <template> [-n <name>]",
		},
		{
			Name:        "destroy",
			Description: "Destroy a dev space",
			Category:    ADM,
			Action:      handlers.Destroy,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:     "name",
					Aliases:  []string{"n"},
					Usage:    "The name of the dev-space",
					Required: true,
				},
			},
			Usage: "-n <name>",
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
