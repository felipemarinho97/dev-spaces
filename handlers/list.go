package handlers

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/felipemarinho97/dev-spaces/helpers"
	"github.com/olekukonko/tablewriter"
)

type OutputFormat string

const (
	OutputFormatWide  OutputFormat = "wide"
	OutputFormatShort OutputFormat = "short"
)

type ListOptions struct {
	Output OutputFormat
}

func (h *Handler) ListTemplates(ctx context.Context, opts ListOptions) error {
	output := opts.Output
	client := h.EC2Client

	launchTemplates, err := helpers.GetLaunchTemplates(ctx, client)
	if err != nil {
		return err
	}

	managedInstances, err := helpers.GetManagedInstances(ctx, client)
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
