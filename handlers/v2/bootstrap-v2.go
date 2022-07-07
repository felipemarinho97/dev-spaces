package v2

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
	DevSpaceAMI          string             `yaml:"devspace_ami" validate:"nonzero"`
	AvailabilityZone     string             `yaml:"availability_zone"`
	PreferedInstanceType types.InstanceType `yaml:"prefered_instance_type"`
	StartupScript        string             `yaml:"startup_script" validate:"nonzero"`
	InstanceProfileArn   string             `yaml:"instance_profile_arn" validate:"nonzero"`
	KeyName              string             `yaml:"key_name" validate:"nonzero"`
	SecurityGroupIds     []string           `yaml:"security_group_ids"`
	StorageSize          int32              `yaml:"storage_size"`
	HostStorageSize      int32              `yaml:"host_storage_size"`
	RootDeviceName       string             `yaml:"-"`
}

type BootstrapSpec struct {
	ec2Client *ec2.Client
	template  *BootstrapTemplate
}

func BootstrapV2(c *cli.Context) error {
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

	if name == "" && b.template.TemplateName != "" {
		name = b.template.TemplateName
	} else if name == "" {
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

	// get the root device name fot this ami
	ami, err := b.ec2Client.DescribeImages(ctx, &ec2.DescribeImagesInput{
		ImageIds: []string{b.template.HostAMI},
	})
	if err != nil {
		return fmt.Errorf("error describing host ami: %v", err)
	}

	rootDeviceName := *ami.Images[0].RootDeviceName
	b.template.RootDeviceName = rootDeviceName

	taskRunner, err := b.createSpotTaskRunner(ctx, name)
	if err != nil {
		return err
	}
	ub.SetDescription(fmt.Sprintf("spot task created: %s - waiting instance to be assigned", *taskRunner.SpotFleetRequestId))
	id, err := b.waitForInstance(ctx, name, *taskRunner.SpotFleetRequestId, types.InstanceStateNameRunning)
	ub.SetDescription(fmt.Sprintf("instance created: %s", id))
	if err != nil {
		return err
	}

	// get the volume id associated with the instance
	out, err := b.ec2Client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
		InstanceIds: []string{id},
	})
	if err != nil {
		return err
	}
	volumeId := out.Reservations[0].Instances[0].BlockDeviceMappings[0].Ebs.VolumeId
	volumeZone := out.Reservations[0].Instances[0].Placement.AvailabilityZone
	b.template.AvailabilityZone = *volumeZone

	ub.SetDescription(fmt.Sprintf("tagging volume: %s", *volumeId))
	// tag the volume
	_, err = b.ec2Client.CreateTags(ctx, &ec2.CreateTagsInput{
		Resources: []string{*volumeId},
		Tags:      util.GenerateTags(name),
	})
	if err != nil {
		return err
	}

	// inject the volume id into the startup script
	b.template.StartupScript = strings.Replace(b.template.StartupScript, "{{volume_id}}", *volumeId, 1)

	// // wait for the instance to become healthy
	// ub.SetDescription(fmt.Sprintf("waiting for instance to be healthy: %s", id))
	// err = b.waitForInstanceHealthy(ctx, id)
	// if err != nil {
	// 	return err
	// }

	// cancel the spot task
	ub.SetDescription(fmt.Sprintf("stopping instance: %s", id))
	_, err = b.ec2Client.CancelSpotFleetRequests(ctx, &ec2.CancelSpotFleetRequestsInput{
		SpotFleetRequestIds: []string{*taskRunner.SpotFleetRequestId},
		TerminateInstances:  aws.Bool(true),
	})
	if err != nil {
		return err
	}

	ub.SetDescription(fmt.Sprintf("waiting for instance=%s to finish - this may take a few minutes", id))
	id, err = b.waitForInstance(ctx, name, *taskRunner.SpotFleetRequestId, types.InstanceStateNameTerminated)
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
					DeviceName: &b.template.RootDeviceName,
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
				Tags:         append(util.GenerateTags(name), types.Tag{Key: aws.String("dev-spaces:zone"), Value: &b.template.AvailabilityZone}),
			},
		},
	})
	if err != nil {
		return nil, err
	}

	return o.LaunchTemplate, nil
}

func (b *BootstrapSpec) createSpotTaskRunner(ctx context.Context, name string) (*ec2.RequestSpotFleetOutput, error) {
	input := &ec2.RequestSpotFleetInput{
		SpotFleetRequestConfig: &types.SpotFleetRequestConfigData{
			TargetCapacity: aws.Int32(1),
			ClientToken:    aws.String(uuid.NewV4().String()),
			Type:           types.FleetTypeRequest,
			IamFleetRole:   aws.String("arn:aws:iam::568126575653:role/aws-ec2-spot-fleet-tagging-role"),
			LaunchSpecifications: []types.SpotFleetLaunchSpecification{
				{
					ImageId:      &b.template.DevSpaceAMI,
					InstanceType: types.InstanceType(b.template.PreferedInstanceType),
					KeyName:      &b.template.KeyName,
					IamInstanceProfile: &types.IamInstanceProfileSpecification{
						Arn: &b.template.InstanceProfileArn,
					},
					BlockDeviceMappings: []types.BlockDeviceMapping{
						{
							DeviceName: &b.template.RootDeviceName,
							Ebs: &types.EbsBlockDevice{
								DeleteOnTermination: aws.Bool(false),
								Encrypted:           aws.Bool(true),
								Iops:                aws.Int32(3000),
								Throughput:          aws.Int32(125),
								VolumeSize:          aws.Int32(b.template.StorageSize),
								VolumeType:          types.VolumeTypeGp3,
							},
						},
					},
					TagSpecifications: []types.SpotFleetTagSpecification{
						{
							ResourceType: types.ResourceTypeInstance,
							Tags:         util.GenerateTags(name),
						},
					},
				},
			},
			TagSpecifications: []types.TagSpecification{
				{
					ResourceType: types.ResourceTypeSpotFleetRequest,
					Tags:         util.GenerateTags(name),
				},
			},
		},
	}

	out, err := b.ec2Client.RequestSpotFleet(ctx, input)
	if err != nil {
		fmt.Println(err.Error())
		return nil, err
	}

	return out, nil
}

func (b *BootstrapSpec) waitForInstance(ctx context.Context, name, requestID string, state types.InstanceStateName) (string, error) {
	for {
		time.Sleep(time.Second * 1)
		out2, err := b.ec2Client.DescribeSpotFleetInstances(ctx, &ec2.DescribeSpotFleetInstancesInput{
			SpotFleetRequestId: &requestID,
		})
		if err != nil {
			fmt.Println(err)
			continue
		}

		if len(out2.ActiveInstances) == 0 && state == types.InstanceStateNameTerminated {
			return "", nil
		}

		time.Sleep(time.Second * 1)
		for _, s := range out2.ActiveInstances {
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

func (b *BootstrapSpec) templateExists(ctx context.Context, name string) (bool, error) {

	// t, err := getLaunchTemplates(ctx, b.ec2Client)
	// if err != nil {
	// 	return false, err
	// }

	// for _, t := range t.LaunchTemplates {
	// 	if *t.LaunchTemplateName == name {
	// 		return true, nil
	// 	}
	// }

	return false, nil
}

func (b *BootstrapSpec) waitForInstanceHealthy(ctx context.Context, instanceID string) error {
	// check status check
	for {
		time.Sleep(time.Second * 1)
		out, err := b.ec2Client.DescribeInstanceStatus(ctx, &ec2.DescribeInstanceStatusInput{
			InstanceIds: []string{instanceID},
		})
		if err != nil {
			fmt.Println(err)
			continue
		}

		for _, s := range out.InstanceStatuses {
			if s.InstanceStatus.Status == types.SummaryStatusOk {
				return nil
			}
		}

	}
}
