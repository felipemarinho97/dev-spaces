package handlers

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/felipemarinho97/dev-spaces/helpers"
	"github.com/felipemarinho97/dev-spaces/util"
	awsUtil "github.com/felipemarinho97/invest-path/util"
)

type CopyOptions struct {
	// Name of the Dev Space
	Name string `validate:"required"`
	// Region is the new region of the instance
	Region string `validate:"required"`
	// AvailabilityZone is the new availability zone of the instance (optional)
	AvailabilityZone string
}

type CopyOutput struct {
	// LaunchTemplateID of the new launch template
	LaunchTemplateID string
	// VolumeID of the new volume
	VolumeID string
	// Zone of the new instance
	Zone string
}

func (h Handler) Copy(ctx context.Context, opts CopyOptions) (CopyOutput, error) {
	err := util.Validator.Struct(opts)
	if err != nil {
		return CopyOutput{}, err
	}

	// check if it is a copy from the same region
	if opts.Region == h.Region {
		return CopyOutput{}, errors.New("cannot copy from the same region")
	}

	client := h.EC2Client
	config, err := awsUtil.LoadAWSConfig()
	if err != nil {
		return CopyOutput{}, err
	}
	config.Region = opts.Region
	newRegionClient := ec2.NewFromConfig(config)

	name, version := util.GetTemplateNameAndVersion(opts.Name)
	template, err := helpers.GetLaunchTemplateByName(ctx, client, name)
	if err != nil {
		return CopyOutput{}, err
	}

	// check if the space is running
	h.Logger.Debug("checking if the space is running")
	volumeID := util.GetTag(template.Tags, "dev-spaces:volume-id")
	if volumeID == "" {
		return CopyOutput{}, errors.New("unable to find volume ID")
	}
	isAttached, err := helpers.IsEBSAttached(ctx, client, volumeID)
	if err != nil || isAttached {
		if err != nil {
			h.Logger.Error("unable to check if the volume is attached: ", err.Error())
		}
		return CopyOutput{}, errors.New("make sure the dev-space is not running")
	}

	// create a snapshot of the volume
	h.Logger.Info("creating a snapshot of the volume")
	snapshot, err := helpers.CreateSnapshot(ctx, client, volumeID)
	if err != nil {
		return CopyOutput{}, err
	}

	// wait for the snapshot to be available
	h.Logger.Info("waiting for the snapshot to be available")
	err = helpers.WaitForSnapshot(ctx, client, snapshot)
	if err != nil {
		return CopyOutput{}, err
	}

	// copy the snapshot to the new region
	h.Logger.Info("copying the snapshot to the new region")
	copySnapshot, err := newRegionClient.CopySnapshot(ctx, &ec2.CopySnapshotInput{
		SourceSnapshotId: aws.String(snapshot),
		SourceRegion:     aws.String(h.Region),
		Description:      aws.String(fmt.Sprintf("%s-%s", name, version)),
	})
	if err != nil {
		return CopyOutput{}, err
	}

	// wait for the snapshot to be available
	h.Logger.Info("waiting for the snapshot to be available")
	err = helpers.WaitForSnapshot(ctx, newRegionClient, *copySnapshot.SnapshotId)
	if err != nil {
		return CopyOutput{}, err
	}

	// create a new volume from the copied snapshot
	h.Logger.Info("creating a new volume from the copied snapshot")
	newVolume, err := newRegionClient.CreateVolume(ctx, &ec2.CreateVolumeInput{
		AvailabilityZone: aws.String(opts.AvailabilityZone),
		SnapshotId:       copySnapshot.SnapshotId,
		VolumeType:       types.VolumeTypeGp3,
		Iops:             aws.Int32(3000),
		Throughput:       aws.Int32(125),
		TagSpecifications: []types.TagSpecification{
			{
				ResourceType: types.ResourceTypeVolume,
				Tags:         util.GenerateTags(name),
			},
		},
	})
	if err != nil {
		return CopyOutput{}, err
	}

	// get default launch template version
	h.Logger.Debug("getting default launch template version")
	defaultVersion, err := helpers.GetDefaultLaunchTemplateVersion(ctx, client, *template.LaunchTemplateId)
	if err != nil {
		return CopyOutput{}, err
	}

	// get the architecture of the machine
	h.Logger.Debug("getting the architecture of the machine")
	currentHostImage, err := helpers.GetImage(ctx, client, *defaultVersion.LaunchTemplateData.ImageId)
	hostArchitecture := currentHostImage.Architecture

	// create a new launch template with the same specifications as the old one
	h.Logger.Info("creating a new launch template with the same specifications as the old one")
	instanceProfileArn := ""
	if defaultVersion.LaunchTemplateData.IamInstanceProfile != nil {
		instanceProfileArn = util.GetValue(defaultVersion.LaunchTemplateData.IamInstanceProfile.Arn)
	}

	// decode startup script
	startupScript, err := base64.StdEncoding.DecodeString(util.GetValue(defaultVersion.LaunchTemplateData.UserData))
	if err != nil {
		return CopyOutput{}, err
	}

	// with architecture of the new instance
	hostImage, err := helpers.FindHostAMI(ctx, newRegionClient, hostArchitecture)
	if err != nil {
		return CopyOutput{}, err
	}

	newLaunchTemplate, err := helpers.CreateLaunchTemplate(ctx, newRegionClient, h.Logger, helpers.CreateLaunchTemplateInput{
		Name:               name,
		VolumeId:           *newVolume.VolumeId,
		VolumeZone:         opts.AvailabilityZone,
		StartupScript:      string(startupScript),
		SecurityGroupIds:   []string{},
		KeyName:            *defaultVersion.LaunchTemplateData.KeyName,
		InstanceProfileArn: &instanceProfileArn,
		Host: helpers.CreateLaunchTemplateHost{
			AMIID: *hostImage.ImageId,
			Device: helpers.CreateLaunchTemplateHostDevice{
				Name:       *hostImage.RootDeviceName,
				Size:       *hostImage.BlockDeviceMappings[0].Ebs.VolumeSize,
				Type:       string(hostImage.BlockDeviceMappings[0].Ebs.VolumeType),
				IOPS:       hostImage.BlockDeviceMappings[0].Ebs.Iops,
				Throughput: hostImage.BlockDeviceMappings[0].Ebs.Throughput,
			},
		},
	})
	if err != nil {
		return CopyOutput{}, err
	}

	// delete both snapshots
	h.Logger.Info("deleting snapshots")
	_, err = client.DeleteSnapshot(ctx, &ec2.DeleteSnapshotInput{
		SnapshotId: aws.String(snapshot),
	})
	if err != nil {
		return CopyOutput{}, err
	}
	_, err = newRegionClient.DeleteSnapshot(ctx, &ec2.DeleteSnapshotInput{
		SnapshotId: copySnapshot.SnapshotId,
	})
	if err != nil {
		return CopyOutput{}, err
	}

	return CopyOutput{
		LaunchTemplateID: *newLaunchTemplate.LaunchTemplateId,
		VolumeID:         *newVolume.VolumeId,
		Zone:             *newVolume.AvailabilityZone,
	}, nil
}
