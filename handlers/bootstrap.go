package handlers

import (
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/felipemarinho97/dev-spaces/helpers"
	"github.com/felipemarinho97/dev-spaces/log"
	"github.com/felipemarinho97/dev-spaces/util"
	awsUtil "github.com/felipemarinho97/invest-path/util"
	"github.com/urfave/cli/v2"
	"gopkg.in/validator.v2"
)

type BootstrapTemplate struct {
	TemplateName         string             `yaml:"template_name"`
	HostAMI              string             `yaml:"host_ami" validate:"nonzero"`
	BootstrapAMI         string             `yaml:"bootstrap_ami" validate:"nonzero"`
	AvailabilityZone     string             `yaml:"availability_zone"`
	PreferedInstanceType types.InstanceType `yaml:"prefered_instance_type"`
	BootstrapScript      string             `yaml:"bootstrap_script" validate:"nonzero"`
	StartupScript        string             `yaml:"startup_script" validate:"nonzero"`
	InstanceProfileArn   string             `yaml:"instance_profile_arn" validate:"nonzero"`
	KeyName              string             `yaml:"key_name" validate:"nonzero"`
	SecurityGroupIds     []string           `yaml:"security_group_ids"`
	StorageSize          int32              `yaml:"storage_size"`
	HostStorageSize      int32              `yaml:"host_storage_size"`
}

type BootstrapSpec struct {
	ec2Client *ec2.Client
	template  *BootstrapTemplate
}

func Bootstrap(c *cli.Context) error {
	ctx := c.Context
	name := c.String("name")
	template := c.String("template")
	region := c.String("region")
	ub := util.NewUnknownBar("Bootstrapping")
	ub.Start()

	config, err := awsUtil.LoadAWSConfig()
	config.Region = region
	if err != nil {
		return err
	}

	client := ec2.NewFromConfig(config)

	b := &BootstrapSpec{
		ec2Client: client,
	}
	err = util.LoadYAML(template, &b.template)
	if err != nil {
		return fmt.Errorf("error loading template: %v", err)
	}
	err = validator.Validate(b.template)
	if err != nil {
		return fmt.Errorf("error validating template: %v", err)
	}

	az := b.template.AvailabilityZone
	size := b.template.StorageSize
	if name == "" && b.template.TemplateName != "" {
		name = b.template.TemplateName
	} else {
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

	ub.SetDescription(fmt.Sprintf("creating ebs volume for %s", name))
	volume, err := helpers.CreateEBSVolume(ctx, client, name, size, az)
	if err != nil {
		return err
	}
	ub.SetDescription(fmt.Sprintf("volume created: %s", *volume.VolumeId))
	b.template.BootstrapScript = strings.Replace(b.template.BootstrapScript, "{{volume_id}}", *volume.VolumeId, -1)
	b.template.StartupScript = strings.Replace(b.template.StartupScript, "{{volume_id}}", *volume.VolumeId, -1)

	ub.SetDescription(fmt.Sprintf("creating instance for running bootstrap task: %s", name))

	taskRunner, err := helpers.CreateSpotTaskRunner(ctx, client, helpers.CreateSpotTaskInput{
		Name:               &name,
		DeviceName:         aws.String("/dev/xvda"),
		StorageSize:        &size,
		AMIID:              &b.template.BootstrapAMI,
		KeyName:            &b.template.KeyName,
		InstanceType:       aws.String(string(b.template.PreferedInstanceType)),
		InstanceProfileArn: &b.template.InstanceProfileArn,
		StartupScript:      &b.template.BootstrapScript,
		Zone:               &az,
	})
	if err != nil {
		return err
	}
	ub.SetDescription(fmt.Sprintf("spot task created: %s - waiting instance to be assigned", *taskRunner.SpotFleetRequestId))
	id, err := helpers.WaitForSpotFleetInstance(ctx, client, *taskRunner.SpotFleetRequestId, types.InstanceStateNameRunning)
	ub.SetDescription(fmt.Sprintf("instance created: %s", id))
	if err != nil {
		return err
	}

	ub.SetDescription(fmt.Sprintf("waiting for bootstrap_script on instance=%s to finish - this may take a few minutes", id))
	id, err = helpers.WaitForSpotFleetInstance(ctx, client, *taskRunner.SpotFleetRequestId, types.InstanceStateNameTerminated)
	if err != nil {
		return err
	}
	ub.SetDescription(fmt.Sprintf("Task terminated: %s", id))
	ub.Stop()

	o, err := helpers.CreateLaunchTemplate(ctx, client, log.NewCLILogger(), helpers.CreateLaunchTemplateInput{
		Name:               name,
		VolumeId:           *volume.VolumeId,
		VolumeZone:         az,
		StartupScript:      b.template.StartupScript,
		SecurityGroupIds:   b.template.SecurityGroupIds,
		KeyName:            b.template.KeyName,
		InstanceProfileArn: &b.template.InstanceProfileArn,
		Host: helpers.CreateLaunchTemplateHost{
			AMIID: b.template.HostAMI,
			Device: helpers.CreateLaunchTemplateHostDevice{
				Name:       "/dev/xvda",
				Size:       b.template.HostStorageSize,
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
