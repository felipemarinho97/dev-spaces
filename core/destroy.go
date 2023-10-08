package core

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/felipemarinho97/dev-spaces/core/helpers"
	"github.com/felipemarinho97/dev-spaces/core/log"
	"github.com/felipemarinho97/dev-spaces/core/util"
	"github.com/felipemarinho97/invest-path/clients"
)

type DestroySpec struct {
	ec2Client clients.IEC2Client
	log       log.Logger
}

type DestroyOptions struct {
	Name string `validate:"required"`
}

func (h *Handler) Destroy(ctx context.Context, opts DestroyOptions) error {
	err := util.Validator.Struct(opts)
	if err != nil {
		return err
	}

	name := opts.Name
	client := h.EC2Client
	log := h.Logger

	ds := &DestroySpec{
		ec2Client: client,
		log:       log,
	}

	// Destroy spot requests
	_, err = helpers.CancelSpotRequests(ctx, client, log, name)
	if err != nil {
		log.Error(err.Error())
	}

	// Destroy security groups
	err = ds.destroySecurityGroups(ctx, name)
	if err != nil {
		log.Error(err.Error())
	}

	// Destroy launch templates
	err = ds.destroyLaunchTemplate(ctx, name)
	if err != nil {
		log.Error(err.Error())
	}

	// Destroy all the created volumes for this template
	err = ds.destroyVolumes(ctx, name)
	if err != nil {
		log.Error(err.Error())
	}

	return nil
}

func (ds *DestroySpec) destroyVolumes(ctx context.Context, templateName string) error {
	volumes, err := ds.getVolumes(ctx, templateName)
	if err != nil {
		return err
	}

	for _, volume := range volumes {
		// wait until ebs volume is detached
		helpers.WaitUntilEBSUnattached(ctx, ds.ec2Client, *volume.VolumeId)
		ds.log.Info(fmt.Sprintf("Destroying volume %s", *volume.VolumeId))
		_, err := ds.ec2Client.DeleteVolume(ctx, &ec2.DeleteVolumeInput{
			VolumeId: volume.VolumeId,
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func (ds *DestroySpec) getVolumes(ctx context.Context, templateName string) ([]types.Volume, error) {
	volumes, err := ds.ec2Client.DescribeVolumes(ctx, &ec2.DescribeVolumesInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("tag:managed-by"),
				Values: []string{"dev-spaces"},
			},
			{
				Name:   aws.String("tag:dev-spaces:name"),
				Values: []string{templateName},
			},
		},
	})
	if err != nil {
		return nil, err
	}

	return volumes.Volumes, nil
}

func (ds *DestroySpec) destroySecurityGroups(ctx context.Context, templateName string) error {
	securityGroups, err := ds.getSecurityGroups(ctx, templateName)
	if err != nil {
		return err
	}

	for _, securityGroup := range securityGroups {
		ds.log.Info(fmt.Sprintf("Destroying security group %s", *securityGroup.GroupId))
		_, err := ds.ec2Client.DeleteSecurityGroup(ctx, &ec2.DeleteSecurityGroupInput{
			GroupId: securityGroup.GroupId,
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func (ds *DestroySpec) getSecurityGroups(ctx context.Context, templateName string) ([]types.SecurityGroup, error) {
	securityGroups, err := ds.ec2Client.DescribeSecurityGroups(ctx, &ec2.DescribeSecurityGroupsInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("tag:managed-by"),
				Values: []string{"dev-spaces"},
			},
			{
				Name:   aws.String("tag:dev-spaces:name"),
				Values: []string{templateName},
			},
		},
	})
	if err != nil {
		return nil, err
	}

	return securityGroups.SecurityGroups, nil
}

func (ds *DestroySpec) destroyLaunchTemplate(ctx context.Context, templateName string) error {
	launchTemplates, err := ds.getLaunchTemplate(ctx, templateName)
	if err != nil {
		return err
	}

	for _, launchTemplate := range launchTemplates {
		ds.log.Info(fmt.Sprintf("Destroying launch template %s", *launchTemplate.LaunchTemplateId))
		_, err = ds.ec2Client.DeleteLaunchTemplate(ctx, &ec2.DeleteLaunchTemplateInput{
			LaunchTemplateId: launchTemplate.LaunchTemplateId,
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func (ds *DestroySpec) getLaunchTemplate(ctx context.Context, templateName string) ([]types.LaunchTemplate, error) {
	launchTemplates, err := ds.ec2Client.DescribeLaunchTemplates(ctx, &ec2.DescribeLaunchTemplatesInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("tag:managed-by"),
				Values: []string{"dev-spaces"},
			},
			{
				Name:   aws.String("tag:dev-spaces:name"),
				Values: []string{templateName},
			},
		},
	})
	if err != nil {
		return nil, err
	}

	return launchTemplates.LaunchTemplates, nil
}
