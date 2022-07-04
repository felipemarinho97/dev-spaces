package handlers

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/felipemarinho97/dev-spaces/util"
	awsUtil "github.com/felipemarinho97/invest-path/util"
	uuid "github.com/satori/go.uuid"
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
	templateExists, err := b.templateExists(ctx, name)
	if err != nil {
		return err
	}
	if templateExists {
		return fmt.Errorf("launch template with name %s already exists", name)
	}

	ub.SetDescription(fmt.Sprintf("creating ebs volume for %s", name))
	volume, err := b.createEBSVolume(ctx, az, size, name)
	if err != nil {
		return err
	}
	ub.SetDescription(fmt.Sprintf("volume created: %s", *volume.VolumeId))
	b.template.BootstrapScript = strings.Replace(b.template.BootstrapScript, "{{volume_id}}", *volume.VolumeId, -1)
	b.template.StartupScript = strings.Replace(b.template.StartupScript, "{{volume_id}}", *volume.VolumeId, -1)

	ub.SetDescription(fmt.Sprintf("creating instance for running bootstrap task: %s", name))

	taskRunner, err := b.createSpotTaskRunner(ctx, name)
	if err != nil {
		return err
	}
	ub.SetDescription(fmt.Sprintf("spot task created: %s - waiting instance to be assigned", *taskRunner.SpotInstanceRequests[0].SpotInstanceRequestId))
	id, err := b.waitForInstance(ctx, name, *taskRunner.SpotInstanceRequests[0].SpotInstanceRequestId, types.InstanceStateNameRunning)
	ub.SetDescription(fmt.Sprintf("instance created: %s", id))
	if err != nil {
		return err
	}

	ub.SetDescription(fmt.Sprintf("waiting for bootstrap_script on instance=%s to finish - this may take a few minutes", id))
	id, err = b.waitForInstance(ctx, name, *taskRunner.SpotInstanceRequests[0].SpotInstanceRequestId, types.InstanceStateNameTerminated)
	if err != nil {
		return err
	}
	ub.SetDescription(fmt.Sprintf("Task terminated: %s", id))
	ub.Stop()

	o, err := b.createLaunchTemplate(ctx, name)
	if err != nil {
		return err
	}
	ub.SetDescription(fmt.Sprintf("launch template created: %s", *o.LaunchTemplateId))

	return nil
}

func (b *BootstrapSpec) createSecurityGroup(ctx context.Context, name string) (*string, error) {
	// get th default vpc id
	vpc, err := b.ec2Client.DescribeVpcs(ctx, &ec2.DescribeVpcsInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("is-default"),
				Values: []string{"true"},
			},
		},
	})
	if err != nil {
		return nil, err
	}

	// create security group
	out, err := b.ec2Client.CreateSecurityGroup(ctx, &ec2.CreateSecurityGroupInput{
		GroupName:   aws.String(name),
		Description: aws.String(fmt.Sprintf("Security group for dev-space %s", name)),
		TagSpecifications: []types.TagSpecification{
			{
				ResourceType: types.ResourceTypeSecurityGroup,
				Tags:         util.GenerateTags(name),
			},
		},
		VpcId: vpc.Vpcs[0].VpcId,
	})
	if err != nil {
		return nil, err
	}

	// add ingress rules for ssh (22,2222) from anywhere
	_, err = b.ec2Client.AuthorizeSecurityGroupIngress(ctx, &ec2.AuthorizeSecurityGroupIngressInput{
		GroupId: out.GroupId,
		IpPermissions: []types.IpPermission{
			{
				IpProtocol: aws.String("tcp"),
				FromPort:   aws.Int32(22),
				ToPort:     aws.Int32(22),
				IpRanges: []types.IpRange{
					{
						Description: aws.String("Allow SSH from anywhere"),
						CidrIp:      aws.String("0.0.0.0/0"),
					},
				},
			},
			{
				IpProtocol: aws.String("tcp"),
				FromPort:   aws.Int32(2222),
				ToPort:     aws.Int32(2222),
				IpRanges: []types.IpRange{
					{
						Description: aws.String("Allow SSH from anywhere"),
						CidrIp:      aws.String("0.0.0.0/0"),
					},
				},
			},
		},
	})
	if err != nil {
		return nil, err
	}

	return out.GroupId, nil
}

func (b *BootstrapSpec) createLaunchTemplate(ctx context.Context, name string) (*types.LaunchTemplate, error) {
	dataScript := base64.StdEncoding.EncodeToString([]byte(b.template.StartupScript))

	// create security group
	groupId, err := b.createSecurityGroup(ctx, name)
	if err != nil {
		return nil, err
	}

	o, err := b.ec2Client.CreateLaunchTemplate(ctx, &ec2.CreateLaunchTemplateInput{
		LaunchTemplateName: aws.String(name),
		ClientToken:        aws.String(uuid.NewV4().String()),
		LaunchTemplateData: &types.RequestLaunchTemplateData{
			KeyName: &b.template.KeyName,
			ImageId: &b.template.HostAMI,
			IamInstanceProfile: &types.LaunchTemplateIamInstanceProfileSpecificationRequest{
				Arn: &b.template.InstanceProfileArn,
			},
			Placement: &types.LaunchTemplatePlacementRequest{
				AvailabilityZone: &b.template.AvailabilityZone,
			},
			UserData:         &dataScript,
			SecurityGroupIds: []string{*groupId},
			BlockDeviceMappings: []types.LaunchTemplateBlockDeviceMappingRequest{
				{
					DeviceName: aws.String("/dev/xvda"),
					Ebs: &types.LaunchTemplateEbsBlockDeviceRequest{
						DeleteOnTermination: aws.Bool(true),
						VolumeSize:          &b.template.HostStorageSize,
						VolumeType:          types.VolumeTypeGp3,
						Encrypted:           aws.Bool(true),
						Iops:                aws.Int32(3000),
						Throughput:          aws.Int32(125),
					},
				},
			},
			TagSpecifications: []types.LaunchTemplateTagSpecificationRequest{
				{
					ResourceType: types.ResourceTypeInstance,
					Tags:         util.GenerateTags(name),
				},
				{
					ResourceType: types.ResourceTypeVolume,
					Tags:         util.GenerateTags(name),
				},
			},
		},
		TagSpecifications: []types.TagSpecification{
			{
				ResourceType: types.ResourceTypeLaunchTemplate,
				Tags:         util.GenerateTags(name),
			},
		},
	})
	if err != nil {
		return nil, err
	}

	return o.LaunchTemplate, nil
}

func (b *BootstrapSpec) createEBSVolume(ctx context.Context, az string, size int32, name string) (*ec2.CreateVolumeOutput, error) {
	out, err := b.ec2Client.CreateVolume(ctx, &ec2.CreateVolumeInput{
		AvailabilityZone: &az,
		Size:             &size,
		VolumeType:       types.VolumeTypeGp3,
		ClientToken:      aws.String(uuid.NewV4().String()),
		TagSpecifications: []types.TagSpecification{
			{
				ResourceType: "volume",
				Tags:         util.GenerateTags(name),
			},
		},
		Encrypted:  aws.Bool(true),
		Throughput: aws.Int32(125),
		Iops:       aws.Int32(3000),
	})
	if err != nil {
		fmt.Println(err.Error())
		return nil, err
	}

	return out, nil
}

func (b *BootstrapSpec) createSpotTaskRunner(ctx context.Context, name string) (*ec2.RequestSpotInstancesOutput, error) {
	dataScript := base64.StdEncoding.EncodeToString([]byte(b.template.BootstrapScript + "\npoweroff\n"))

	input := &ec2.RequestSpotInstancesInput{
		LaunchSpecification: &types.RequestSpotLaunchSpecification{
			ImageId:      &b.template.BootstrapAMI,
			InstanceType: b.template.PreferedInstanceType,
			UserData:     &dataScript,
			IamInstanceProfile: &types.IamInstanceProfileSpecification{
				Arn: &b.template.InstanceProfileArn,
			},
			Placement: &types.SpotPlacement{
				AvailabilityZone: &b.template.AvailabilityZone,
			},
			KeyName:          &b.template.KeyName,
			SecurityGroupIds: b.template.SecurityGroupIds,
		},
		ClientToken: aws.String(uuid.NewV4().String()),
		TagSpecifications: []types.TagSpecification{
			{
				ResourceType: types.ResourceTypeSpotInstancesRequest,
				Tags:         util.GenerateTags(name),
			},
		},
	}

	if b.template.AvailabilityZone != "" {
		input.AvailabilityZoneGroup = &b.template.AvailabilityZone
	}

	out, err := b.ec2Client.RequestSpotInstances(ctx, input)
	if err != nil {
		fmt.Println(err.Error())
		return nil, err
	}

	return out, nil
}

func (b *BootstrapSpec) waitForInstance(ctx context.Context, name, requestID string, state types.InstanceStateName) (string, error) {
	for {
		time.Sleep(time.Second * 1)
		out2, err := b.ec2Client.DescribeSpotInstanceRequests(ctx, &ec2.DescribeSpotInstanceRequestsInput{
			SpotInstanceRequestIds: []string{requestID},
		})
		if err != nil {
			fmt.Println(err)
			continue
		}

		time.Sleep(time.Second * 1)
		for _, s := range out2.SpotInstanceRequests {
			if s.State == types.SpotInstanceStateActive && util.IsDevSpace(s.Tags, name) && util.IsManaged(s.Tags) {
				out, err := b.ec2Client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
					InstanceIds: []string{*s.InstanceId},
				})
				if err != nil {
					fmt.Println(err)
					continue
				}

				for _, i := range out.Reservations {
					for _, r := range i.Instances {
						if r.State.Name == state {
							return *r.InstanceId, nil
						}
					}
				}
			}
		}
	}
}

func (b *BootstrapSpec) templateExists(ctx context.Context, name string) (bool, error) {

	t, err := getLaunchTemplates(ctx, b.ec2Client)
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
