package helpers

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/felipemarinho97/dev-spaces/log"
	"github.com/felipemarinho97/dev-spaces/util"
	"github.com/felipemarinho97/invest-path/clients"
	uuid "github.com/satori/go.uuid"
)

type CreateLaunchTemplateInput struct {
	Name               string
	VolumeId           string
	VolumeZone         string
	StartupScript      string
	SecurityGroupIds   []string
	KeyName            string
	InstanceProfileArn *string
	Host               CreateLaunchTemplateHost
}

type CreateLaunchTemplateHost struct {
	AMIID  string
	Device CreateLaunchTemplateHostDevice
}

type CreateLaunchTemplateHostDevice struct {
	Name       string
	Size       int32
	Type       string
	IOPS       *int32
	Throughput *int32
}

func CreateLaunchTemplate(ctx context.Context, ec2Client clients.IEC2Client, log log.Logger, in CreateLaunchTemplateInput) (*types.LaunchTemplate, error) {
	dataScript := base64.StdEncoding.EncodeToString([]byte(in.StartupScript))

	// create security group
	groupId, err := CreateSecurityGroup(ctx, ec2Client, log, in.Name)
	if err != nil {
		return nil, err
	}

	in.SecurityGroupIds = append(in.SecurityGroupIds, *groupId)

	// lauch template data
	ltd := &types.RequestLaunchTemplateData{
		KeyName: &in.KeyName,
		ImageId: &in.Host.AMIID,
		Placement: &types.LaunchTemplatePlacementRequest{
			AvailabilityZone: &in.VolumeZone,
		},
		UserData:         &dataScript,
		SecurityGroupIds: in.SecurityGroupIds,
		BlockDeviceMappings: []types.LaunchTemplateBlockDeviceMappingRequest{
			{
				DeviceName: &in.Host.Device.Name,
				Ebs: &types.LaunchTemplateEbsBlockDeviceRequest{
					DeleteOnTermination: aws.Bool(true),
					Encrypted:           aws.Bool(true),
					VolumeSize:          &in.Host.Device.Size,
					VolumeType:          types.VolumeType(in.Host.Device.Type),
					Iops:                in.Host.Device.IOPS,
					Throughput:          in.Host.Device.Throughput,
				},
			},
		},
		TagSpecifications: []types.LaunchTemplateTagSpecificationRequest{
			{
				ResourceType: types.ResourceTypeInstance,
				Tags:         util.GenerateTags(in.Name),
			},
			{
				ResourceType: types.ResourceTypeVolume,
				Tags:         util.GenerateTags(in.Name),
			},
		},
	}

	if in.InstanceProfileArn != nil && *in.InstanceProfileArn != "" {
		ltd.IamInstanceProfile = &types.LaunchTemplateIamInstanceProfileSpecificationRequest{
			Arn: in.InstanceProfileArn,
		}
	}

	// create launch template
	log.Info("Creating launch template..")
	o, err := ec2Client.CreateLaunchTemplate(ctx, &ec2.CreateLaunchTemplateInput{
		LaunchTemplateName: aws.String(in.Name),
		ClientToken:        aws.String(uuid.NewV4().String()),
		LaunchTemplateData: ltd,
		TagSpecifications: []types.TagSpecification{
			{
				ResourceType: types.ResourceTypeLaunchTemplate,
				Tags: append(
					util.GenerateTags(in.Name),
					types.Tag{
						Key:   aws.String("dev-spaces:zone"),
						Value: &in.VolumeZone,
					},
					types.Tag{
						Key:   aws.String("dev-spaces:volume-id"),
						Value: &in.VolumeId,
					},
				),
			},
		},
	})
	if err != nil {
		return nil, err
	}

	return o.LaunchTemplate, nil
}

func GetDefaultLaunchTemplateVersion(ctx context.Context, client clients.IEC2Client, templateID string) (*types.LaunchTemplateVersion, error) {
	launchTemplate, err := client.DescribeLaunchTemplateVersions(ctx, &ec2.DescribeLaunchTemplateVersionsInput{
		LaunchTemplateId: aws.String(templateID),
		Versions:         []string{"$Default"},
	})
	if err != nil {
		return nil, err
	}

	if len(launchTemplate.LaunchTemplateVersions) == 0 {
		return nil, errors.New("unable to find default launch template version")
	}

	return &launchTemplate.LaunchTemplateVersions[0], nil
}

func GetLaunchTemplates(ctx context.Context, client clients.IEC2Client) (*ec2.DescribeLaunchTemplatesOutput, error) {
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

func GetLaunchTemplateByName(ctx context.Context, client clients.IEC2Client, name string) (*types.LaunchTemplate, error) {
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

func TemplateExists(ctx context.Context, client clients.IEC2Client, name string) (bool, error) {
	t, err := GetLaunchTemplates(ctx, client)
	if err != nil {
		return false, err
	}

	for _, t := range t.LaunchTemplates {
		if *t.LaunchTemplateName == name {
			return true, nil
		}
	}

	return false, nil
}

func DeleteLaunchTemplate(ctx context.Context, client clients.IEC2Client, name string) error {
	_, err := client.DeleteLaunchTemplate(ctx, &ec2.DeleteLaunchTemplateInput{
		LaunchTemplateName: aws.String(name),
	})
	return err
}
