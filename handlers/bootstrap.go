package handlers

import (
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/felipemarinho97/dev-spaces/helpers"
	"github.com/felipemarinho97/dev-spaces/log"
	"github.com/felipemarinho97/dev-spaces/util"
	awsUtil "github.com/felipemarinho97/invest-path/util"
	"github.com/urfave/cli/v2"
)

type BootstrapTemplate struct {
	TemplateName          string              `yaml:"template_name"`
	HostAMI               AMIFilter           `yaml:"host_ami" validate:"required"`
	BootstrapAMI          AMIFilter           `yaml:"bootstrap_ami" validate:"required"`
	AvailabilityZone      string              `yaml:"availability_zone"`
	PreferedInstanceSpecs PreferedLaunchSpecs `yaml:"prefered_instance_specs"`
	BootstrapScript       string              `yaml:"bootstrap_script" validate:"required"`
	StartupScript         string              `yaml:"startup_script" validate:"required"`
	InstanceProfileArn    string              `yaml:"instance_profile_arn"`
	KeyName               string              `yaml:"key_name" validate:"required"`
	SecurityGroupIds      []string            `yaml:"security_group_ids"`
	StorageSize           int32               `yaml:"storage_size"`
}

func Bootstrap(c *cli.Context) error {
	ctx := c.Context
	name := c.String("name")
	templatePath := c.String("template")
	region := c.String("region")
	ub := util.NewUnknownBar("Bootstrapping")
	ub.Start()

	config, err := awsUtil.LoadAWSConfig()
	config.Region = region
	if err != nil {
		return err
	}

	client := ec2.NewFromConfig(config)

	var template BootstrapTemplate
	err = util.LoadYAML(templatePath, &template)
	if err != nil {
		return fmt.Errorf("error loading template: %v", err)
	}
	err = util.Validator.Struct(template)
	if err != nil {
		return fmt.Errorf("error validating template: %v", err)
	}

	az := template.AvailabilityZone
	if name == "" && template.TemplateName != "" {
		name = template.TemplateName
	} else if name == "" {
		return fmt.Errorf("flag name or template_name must be provided")
	}

	// check if a launch template with the same name already exists
	templateExists, err := helpers.TemplateExists(ctx, client, name)
	if err != nil {
		return err
	}
	if templateExists {
		return fmt.Errorf("launch template with name %s already exists", name)
	}

	// get host and dev space ami
	hostAMI, err := helpers.GetImageFromFilter(ctx, client, helpers.AMIFilter{
		Name:  template.HostAMI.Name,
		ID:    template.HostAMI.ID,
		Arch:  template.HostAMI.Arch,
		Owner: template.HostAMI.Owner,
	})
	if err != nil {
		return err
	}

	bootstrapAMI, err := helpers.GetImageFromFilter(ctx, client, helpers.AMIFilter{
		Name:  template.BootstrapAMI.Name,
		ID:    template.BootstrapAMI.ID,
		Arch:  template.BootstrapAMI.Arch,
		Owner: template.BootstrapAMI.Owner,
	})
	if err != nil {
		return err
	}

	// check if architecture is compatible with the host ami
	if hostAMI.Architecture != bootstrapAMI.Architecture {
		return fmt.Errorf("host ami architecture %s is not compatible with bootstrap ami architecture %s", hostAMI.Architecture, bootstrapAMI.Architecture)
	}

	ub.SetDescription(fmt.Sprintf("creating instance for running bootstrap task: %s", name))
	taskRunner, err := helpers.CreateSpotTaskRunner(ctx, client, helpers.CreateSpotTaskInput{
		Name:        &name,
		DeviceName:  aws.String("/dev/xvda"),
		StorageSize: bootstrapAMI.BlockDeviceMappings[0].Ebs.VolumeSize,
		AMIID:       bootstrapAMI.ImageId,
		KeyName:     &template.KeyName,
		PreferedLaunchSpecs: &helpers.PreferedLaunchSpecs{
			InstanceType: string(template.PreferedInstanceSpecs.InstanceType),
			MinMemory:    template.PreferedInstanceSpecs.MinMemory,
			MinCPU:       template.PreferedInstanceSpecs.MinCPU,
		},
		InstanceProfileArn: &template.InstanceProfileArn,
		StartupScript:      aws.String(template.BootstrapScript + "\npoweroff"),
		Zone:               &az,
	})
	if err != nil {
		return err
	}
	ub.SetDescription(fmt.Sprintf("spot task created: %s - waiting instance to be assigned", *taskRunner.FleetId))
	id, err := helpers.WaitForFleetInstance(ctx, client, *taskRunner.FleetId, types.InstanceStateNameRunning)
	ub.SetDescription(fmt.Sprintf("instance created: %s", id))
	if err != nil {
		return err
	}

	// get instance zone
	instance, err := helpers.GetInstanceData(ctx, client, id)
	if err != nil {
		return err
	}
	az = *instance.Placement.AvailabilityZone
	ub.SetDescription(fmt.Sprintf("instance created on zone: %s", az))

	ub.SetDescription(fmt.Sprintf("creating ebs volume for %s", name))
	volume, err := helpers.CreateEBSVolume(ctx, client, name, template.StorageSize, az)
	if err != nil {
		return err
	}
	ub.SetDescription(fmt.Sprintf("volume created: %s", *volume.VolumeId))

	// wait for volume to be available
	ub.SetDescription(fmt.Sprintf("waiting for volume %s to be available", *volume.VolumeId))
	err = helpers.WaitForEBSVolume(ctx, client, *volume.VolumeId, types.VolumeStateAvailable)
	if err != nil {
		return err
	}

	// attach volume to instance
	ub.SetDescription(fmt.Sprintf("attaching volume %s to instance %s", *volume.VolumeId, id))
	err = helpers.AttachEBSVolume(ctx, client, id, *volume.VolumeId)
	if err != nil {
		return err
	}

	ub.SetDescription(fmt.Sprintf("waiting for bootstrap_script on instance=%s to finish - this may take a few minutes", id))
	id, err = helpers.WaitForFleetInstance(ctx, client, *taskRunner.FleetId, types.InstanceStateNameTerminated)
	if err != nil {
		return err
	}
	ub.SetDescription(fmt.Sprintf("Task terminated: %s", id))
	ub.Stop()

	hostStorageSize := *hostAMI.BlockDeviceMappings[0].Ebs.VolumeSize

	o, err := helpers.CreateLaunchTemplate(ctx, client, log.NewCLILogger(), helpers.CreateLaunchTemplateInput{
		Name:               name,
		VolumeId:           *volume.VolumeId,
		VolumeZone:         az,
		StartupScript:      template.StartupScript,
		SecurityGroupIds:   template.SecurityGroupIds,
		KeyName:            template.KeyName,
		InstanceProfileArn: &template.InstanceProfileArn,
		Host: helpers.CreateLaunchTemplateHost{
			AMIID: *hostAMI.ImageId,
			Device: helpers.CreateLaunchTemplateHostDevice{
				Name:       "/dev/xvda",
				Size:       hostStorageSize,
				Type:       "gp3",
				IOPS:       aws.Int32(3000),
				Throughput: aws.Int32(125),
			},
		},
	})
	if err != nil {
		return err
	}
	ub.SetDescription(fmt.Sprintf("launch template created: %s", *o.LaunchTemplateId))

	return nil
}
