package handlers

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/felipemarinho97/dev-spaces/util"
	awsUtil "github.com/felipemarinho97/invest-path/util"
	"github.com/olekukonko/tablewriter"
	"github.com/urfave/cli/v2"
)

func ListTemplates(c *cli.Context) error {
	ctx := c.Context
	region := c.String("region")
	output := c.String("output")

	config, err := awsUtil.LoadAWSConfig()
	config.Region = region
	if err != nil {
		return err
	}

	client := ec2.NewFromConfig(config)

	launchTemplates, err := GetLaunchTemplates(ctx, client)
	if err != nil {
		return err
	}

	managedInstances, err := getManagedInstances(ctx, client)
	if err != nil {
		return err
	}

	table := tablewriter.NewWriter(os.Stdout)
	header := []string{"Space Name", "Ver", "ID", "Create Time"}
	if output == "wide" {
		extra_headers := []string{"Instance ID", "Instance Type", "Instance State", "Public DNS", "Public IP", "Key Name", "Zone"}
		header = append(header, extra_headers...)
	}
	table.SetHeader(header)
	table.SetAutoWrapText(false)
	table.SetAutoFormatHeaders(true)
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetCenterSeparator("")
	table.SetColumnSeparator("")
	table.SetRowSeparator("")
	table.SetHeaderLine(true)
	table.SetBorder(false)
	table.SetTablePadding("\t") // pad with tabs
	table.SetNoWhiteSpace(true)

	for _, launchTemplate := range launchTemplates.LaunchTemplates {
		version := *launchTemplate.DefaultVersionNumber
		name := *launchTemplate.LaunchTemplateName
		if version != 1 {
			name = fmt.Sprintf("%s/%d", name, version)
		}

		instance := managedInstances[*launchTemplate.LaunchTemplateName]

		row := []string{
			name,
			fmt.Sprint(*launchTemplate.DefaultVersionNumber),
			*launchTemplate.LaunchTemplateId,
			*aws.String(launchTemplate.CreateTime.Format("2006-01-02 15:04:05")),
		}

		if instance != nil && output == "wide" {
			row = append(row, []string{
				getOrNone(instance.InstanceId),
				fmt.Sprint(instance.InstanceType),
				strings.ToUpper(fmt.Sprint(instance.State.Name)),
				getOrNone(instance.PublicDnsName),
				getOrNone(instance.PublicIpAddress),
				getOrNone(instance.KeyName),
				getOrNone(instance.Placement.AvailabilityZone),
			}...)
		}

		table.Append(row)
	}

	table.Render()

	return nil
}

func getOrNone(v *string) string {
	if v == nil || *v == "" {
		return "-"
	}
	return fmt.Sprint(*v)
}

func GetLaunchTemplates(ctx context.Context, client *ec2.Client) (*ec2.DescribeLaunchTemplatesOutput, error) {
	launchTemplates, err := client.DescribeLaunchTemplates(ctx, &ec2.DescribeLaunchTemplatesInput{
		Filters: []types.Filter{

			{
				Name:   aws.String("tag:managed-by"),
				Values: []string{"dev-spaces"},
			},
		},
	})
	if err != nil {
		return nil, err
	}

	return launchTemplates, nil
}

func getLaunchTemplateByName(ctx context.Context, client *ec2.Client, name string) (*types.LaunchTemplate, error) {
	launchTemplates, err := GetLaunchTemplates(ctx, client)
	if err != nil {
		return nil, err
	}

	for _, launchTemplate := range launchTemplates.LaunchTemplates {
		if *launchTemplate.LaunchTemplateName == name {
			return &launchTemplate, nil
		}
	}

	return nil, fmt.Errorf("launch template not found")
}

func getManagedInstances(ctx context.Context, client *ec2.Client) (map[string]*types.Instance, error) {
	instances, err := client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("tag:managed-by"),
				Values: []string{"dev-spaces"},
			},
		},
	})
	if err != nil {
		return nil, err
	}

	// map instance by space name
	var managedInstances = make(map[string]*types.Instance)
	for _, instance := range instances.Reservations {
		for _, i := range instance.Instances {
			name := util.GetTag(i.Tags, "dev-spaces:name")

			if inst := managedInstances[name]; inst != nil {
				if inst.LaunchTime.After(*i.LaunchTime) {
					continue
				}
			}
			managedInstances[name] = &i
		}
	}

	return managedInstances, nil
}
