package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/felipemarinho97/dev-spaces/handlers"
	"github.com/felipemarinho97/dev-spaces/log"
	awsUtil "github.com/felipemarinho97/invest-path/util"
	"github.com/urfave/cli/v2"
)

const (
	ADM       = "ADMINISTRATION"
	LIFECYCLE = "DEV-SPACE"
)

func GetCLI() *cli.App {
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
		Before: func(ctx *cli.Context) error {
			return loadClients(ctx)
		},
	}

	app.Commands = []*cli.Command{
		{
			Name:        "start",
			Description: "Starts the dev environment by placing a spot request.",
			Usage:       "-n <name> [-c <min-cpus> -m <min-memory> --max-price <max-price> -t <timeout>]",
			Category:    LIFECYCLE,
			Action:      startCommand,
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
			Action:      stopCommand,
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
			Action:      statusCommand,
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
			Action:   createCommand,
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
					Name:    "instance-profile-arn",
					Aliases: []string{"p"},
					Usage:   "Instance profile ARN (arn:aws:iam::<account-id>:instance-profile/<instance-profile-name>) to use.",
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
				&cli.StringFlag{
					Name:  "custom-host-ami",
					Value: "",
					Usage: "Custom AMI to use for the host - use this flag in combination with --custom-startup-script",
				},
				&cli.PathFlag{
					Name:      "custom-startup-script",
					Value:     "",
					TakesFile: true,
					Usage:     "Custom startup script to use for the host",
				},
				&cli.StringSliceFlag{
					Name:  "security-group-ids",
					Value: &cli.StringSlice{},
					Usage: "A list of security group IDs to use. e.g. --security-group-ids sg-123456789 sg-987654321",
				},
			},
			Usage: "-n <name> -k <key-name> -i <ami> [-p <instance-profile-arn> -s <storage-size> -t <prefered-instance-type>]",
		},
		{
			Name:        "list",
			Description: "List all the dev spaces",
			Category:    LIFECYCLE,
			Action:      listCommand,
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
			Action:      destroyCommand,
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
		{
			Name:        "tools",
			Description: "Tools for configuring the dev space. You can use this sub-commands to change instance type, storage size, dev-space region etc.",
			Aliases:     []string{"cfg"},
			Category:    ADM,
			Subcommands: []*cli.Command{
				{
					Name:        "scale",
					Description: "Scale-up or scale-down specifications of the dev space",
					Action:      editSpecCommand,
					Flags: []cli.Flag{
						&cli.StringFlag{
							Name:     "name",
							Aliases:  []string{"n"},
							Usage:    "The name of the dev-space",
							Required: true,
						},
						&cli.StringFlag{
							Name:     "identity-file",
							Aliases:  []string{"i"},
							Usage:    "The path to the SSH identity file",
							Required: true,
						},
						&cli.IntFlag{
							Name:    "min-cpus",
							Aliases: []string{"c"},
							Usage:   "The minimum number of cpus to use for the instance",
							Value:   0,
						},
						&cli.Float64Flag{
							Name:    "min-memory",
							Aliases: []string{"m"},
							Usage:   "The minimum amount of memory to use for the instance",
							Value:   0,
						},
						&cli.StringFlag{
							Name:  "max-price",
							Usage: "The max price to use for the instance",
							Value: "0.5",
						},
					},
					Usage: "-n <name> -i <identity-file> [-c <min-cpus> -m <min-memory> -p <max-price>]",
				},
				{
					Name:        "copy",
					Description: "Copy a dev space to a new region",
					Action:      copyCommand,
					Flags: []cli.Flag{
						&cli.StringFlag{
							Name:     "name",
							Aliases:  []string{"n"},
							Usage:    "The name of the dev-space",
							Required: true,
						},
						&cli.StringFlag{
							Name:     "new-region",
							Aliases:  []string{"r"},
							Usage:    "The region to copy the dev-space to",
							Required: true,
						},
						&cli.StringFlag{
							Name:     "availability-zone",
							Aliases:  []string{"z"},
							Usage:    "The availability zone to copy the dev-space to",
							Required: true,
						},
					},
					Usage: "-n <name> -r <region> -z <availability-zone>",
				},
			},
		},
	}

	return app
}

func loadClients(c *cli.Context) error {
	config, err := awsUtil.LoadAWSConfig()
	config.Region = c.String("region")
	if err != nil {
		return err
	}

	client := ec2.NewFromConfig(config)
	ssmClient := ssm.NewFromConfig(config)
	logger := log.NewCLILogger()

	handler := handlers.NewHandler(config.Region, client, ssmClient, logger)

	// inject the handler into the context
	c.Context = context.WithValue(c.Context, "handler", handler)

	return nil
}
