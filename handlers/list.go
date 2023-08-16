package handlers

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/felipemarinho97/dev-spaces/helpers"
)

type OutputFormat string

const (
	OutputFormatWide  OutputFormat = "wide"
	OutputFormatShort OutputFormat = "short"
)

type ListOptions struct{}

type ListItem struct {
	Name             string
	Version          int64
	LaunchTemplateID string
	CreateTime       string
	InstanceID       string
	InstanceType     string
	InstanceState    string
	PublicDNS        string
	PublicIP         string
	KeyName          string
	Zone             string
}

func (h *Handler) ListSpaces(ctx context.Context, opts ListOptions) ([]ListItem, error) {
	client := h.EC2Client

	launchTemplates, err := helpers.GetLaunchTemplates(ctx, client)
	if err != nil {
		return nil, err
	}

	managedInstances, err := helpers.GetManagedInstances(ctx, client)
	if err != nil {
		return nil, err
	}

	items := toListItems(launchTemplates.LaunchTemplates, managedInstances)

	return items, nil
}

func getOrNone(v *string) string {
	if v == nil || *v == "" {
		return "-"
	}
	return fmt.Sprint(*v)
}

func toListItems(launchTemplates []types.LaunchTemplate, managedInstances map[string]*types.Instance) []ListItem {
	items := make([]ListItem, 0, len(launchTemplates))
	for _, launchTemplate := range launchTemplates {
		version := *launchTemplate.DefaultVersionNumber
		name := *launchTemplate.LaunchTemplateName
		if version != 1 {
			name = fmt.Sprintf("%s/%d", name, version)
		}

		instance := managedInstances[*launchTemplate.LaunchTemplateName]

		item := ListItem{
			Name:             name,
			Version:          *launchTemplate.DefaultVersionNumber,
			LaunchTemplateID: *launchTemplate.LaunchTemplateId,
			CreateTime:       *aws.String(launchTemplate.CreateTime.Format("2006-01-02 15:04:05")),
		}

		if instance != nil {
			item.InstanceID = getOrNone(instance.InstanceId)
			item.InstanceType = fmt.Sprint(instance.InstanceType)
			item.InstanceState = strings.ToUpper(fmt.Sprint(instance.State.Name))
			item.PublicDNS = getOrNone(instance.PublicDnsName)
			item.PublicIP = getOrNone(instance.PublicIpAddress)
			item.KeyName = getOrNone(instance.KeyName)
			item.Zone = getOrNone(instance.Placement.AvailabilityZone)
		}

		items = append(items, item)
	}

	return items
}
