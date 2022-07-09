package v2

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/felipemarinho97/dev-spaces/handlers"
	"github.com/felipemarinho97/dev-spaces/util"
	awsUtil "github.com/felipemarinho97/invest-path/util"
	uuid "github.com/satori/go.uuid"
	"github.com/urfave/cli/v2"
	"gopkg.in/validator.v2"
)

type BootstrapTemplate struct {
	TemplateName         string             `validate:"nonzero,min=3,max=128"`
	DevSpaceAMIID        string             `validate:"nonzero"`
	PreferedInstanceType types.InstanceType `validate:"nonzero"`
	KeyName              string             `validate:"nonzero"`
	InstanceProfileArn   string
	StartupScript        string
	SecurityGroupIds     []string
	StorageSize          int32
	RootDeviceName       string
	HostStorageSize      int32
	HostArchitecture     types.ArchitectureValues
	HostAMIID            string
	HostAMI              *types.Image
	DevSpaceAMI          *types.Image
}

type Bootstrapper struct {
	ec2Client *ec2.Client
	ssmClient *ssm.Client
	template  *BootstrapTemplate
	ub        *util.UnknownBar
}

func BootstrapV2(c *cli.Context) error {
	ctx := c.Context
	name := c.String("name")
	keyName := c.String("key-name")
	instanceProfileArn := c.String("instance-profile-arn")
	devSpaceAMIID := c.String("ami")
	preferedInstanceType := c.String("prefered-instance-type")
	storageSize := c.Int("storage-size")
	region := c.String("region")
	ub := util.NewUnknownBar("Bootstrapping..")
	ub.Start()
	defer ub.Stop()

	config, err := awsUtil.LoadAWSConfig()
	config.Region = region
	if err != nil {
		return err
	}

	client := ec2.NewFromConfig(config)
	ssmClient := ssm.NewFromConfig(config)

	b := &Bootstrapper{
		ec2Client: client,
		ssmClient: ssmClient,
		ub:        ub,
	}

	b.template = &BootstrapTemplate{
		TemplateName:         name,
		DevSpaceAMIID:        devSpaceAMIID,
		InstanceProfileArn:   instanceProfileArn,
		KeyName:              keyName,
		PreferedInstanceType: types.InstanceType(preferedInstanceType),
		StorageSize:          int32(storageSize),
	}
	err = validator.Validate(b.template)
	if err != nil {
		return fmt.Errorf("error validating template: %v", err)
	}

	// check if a launch template with the same name already exists
	templateExists, err := b.templateExists(ctx, name)
	if err != nil {
		return err
	}
	if templateExists {
		return fmt.Errorf("launch template with name %s already exists", name)
	}

	if b.template.StartupScript == "" {
		b.template.StartupScript = DEFAULT_STATUP_SCRIPT
		ub.SetDescription("Using default startup script...")
	}

	// get the architecture of the machine
	devSpaceHostImage, err := b.ec2Client.DescribeImages(ctx, &ec2.DescribeImagesInput{
		ImageIds: []string{b.template.DevSpaceAMIID},
	})
	if err != nil {
		return fmt.Errorf("error describing host ami: %v", err)
	}
	b.template.DevSpaceAMI = &devSpaceHostImage.Images[0]
	b.template.HostArchitecture = devSpaceHostImage.Images[0].Architecture

	// get the best AMI to use for the devspace host
	hostAMI, err := b.findHostAMI(ctx, b.template.HostArchitecture)
	if err != nil {
		return err
	}
	b.template.HostAMIID = hostAMI

	// get the root device name fot this hostImage
	hostImage, err := b.ec2Client.DescribeImages(ctx, &ec2.DescribeImagesInput{
		ImageIds: []string{b.template.HostAMIID},
	})
	if err != nil {
		return fmt.Errorf("error describing host ami: %v", err)
	}

	b.template.HostAMI = &hostImage.Images[0]
	b.template.RootDeviceName = *hostImage.Images[0].RootDeviceName
	b.template.HostStorageSize = *hostImage.Images[0].BlockDeviceMappings[0].Ebs.VolumeSize

	taskRunner, err := b.createSpotTaskRunner(ctx, name)
	if err != nil {
		return err
	}
	ub.SetDescription(fmt.Sprintf("Spot task created: %s - Waiting instance to be assigned..", *taskRunner.SpotFleetRequestId))
	id, err := b.waitForInstance(ctx, name, *taskRunner.SpotFleetRequestId, types.InstanceStateNameRunning)
	ub.SetDescription(fmt.Sprintf("Instance assigned: %s", id))
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

	// tag the volume
	ub.SetDescription(fmt.Sprintf("Tagging volume: %s", *volumeId))
	_, err = b.ec2Client.CreateTags(ctx, &ec2.CreateTagsInput{
		Resources: []string{*volumeId},
		Tags:      util.GenerateTags(name),
	})
	if err != nil {
		return err
	}

	// cancel the spot task
	ub.SetDescription(fmt.Sprintf("Stopping instance: %s", id))
	_, err = b.ec2Client.CancelSpotFleetRequests(ctx, &ec2.CancelSpotFleetRequestsInput{
		SpotFleetRequestIds: []string{*taskRunner.SpotFleetRequestId},
		TerminateInstances:  aws.Bool(true),
	})
	if err != nil {
		return err
	}

	ub.SetDescription(fmt.Sprintf("Waiting for instance: %s to finish.. This may take a few minutes..", id))
	id, err = b.waitForInstance(ctx, name, *taskRunner.SpotFleetRequestId, types.InstanceStateNameTerminated)
	if err != nil {
		return err
	}

	o, err := b.createLaunchTemplate(ctx, name, *volumeId, *volumeZone)
	if err != nil {
		return err
	}
	ub.SetDescription(fmt.Sprintf("Launch template created: %s", *o.LaunchTemplateId))

	ub.SetDescription(fmt.Sprintf("DevSpace \"%s\" created successfully.", name))

	return nil
}

func (b *Bootstrapper) createSecurityGroup(ctx context.Context, name string) (*string, error) {
	b.ub.SetDescription("Creating security group..")

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
	b.ub.SetDescription("Adding ingress rules for ssh..")
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

func (b *Bootstrapper) createLaunchTemplate(ctx context.Context, name, volumeID, zone string) (*types.LaunchTemplate, error) {
	dataScript := base64.StdEncoding.EncodeToString([]byte(b.template.StartupScript))

	// create security group
	groupId, err := b.createSecurityGroup(ctx, name)
	if err != nil {
		return nil, err
	}

	// create launch template
	b.ub.SetDescription("Creating launch template..")
	o, err := b.ec2Client.CreateLaunchTemplate(ctx, &ec2.CreateLaunchTemplateInput{
		LaunchTemplateName: aws.String(name),
		ClientToken:        aws.String(uuid.NewV4().String()),
		LaunchTemplateData: &types.RequestLaunchTemplateData{
			KeyName: &b.template.KeyName,
			ImageId: &b.template.HostAMIID,
			// IamInstanceProfile: &types.LaunchTemplateIamInstanceProfileSpecificationRequest{
			// 	Arn: &b.template.InstanceProfileArn,
			// },
			Placement: &types.LaunchTemplatePlacementRequest{
				AvailabilityZone: &zone,
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
				Tags: append(
					util.GenerateTags(name),
					types.Tag{
						Key:   aws.String("dev-spaces:zone"),
						Value: &zone,
					},
					types.Tag{
						Key:   aws.String("dev-spaces:volume-id"),
						Value: &volumeID,
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

func (b *Bootstrapper) createSpotTaskRunner(ctx context.Context, name string) (*ec2.RequestSpotFleetOutput, error) {
	minVolumeSize := *b.template.DevSpaceAMI.BlockDeviceMappings[0].Ebs.VolumeSize
	if b.template.StorageSize < minVolumeSize {
		b.template.StorageSize = minVolumeSize
	}

	launchSpecification := types.SpotFleetLaunchSpecification{
		ImageId: &b.template.DevSpaceAMIID,
		KeyName: &b.template.KeyName,
		// IamInstanceProfile: &types.IamInstanceProfileSpecification{
		// 	Arn: &b.template.InstanceProfileArn,
		// },
		BlockDeviceMappings: []types.BlockDeviceMapping{
			{
				DeviceName: b.template.DevSpaceAMI.BlockDeviceMappings[0].DeviceName,
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
	}

	if b.template.PreferedInstanceType != "" {
		launchSpecification.InstanceType = types.InstanceType(b.template.PreferedInstanceType)
	}

	input := &ec2.RequestSpotFleetInput{
		SpotFleetRequestConfig: &types.SpotFleetRequestConfigData{
			TargetCapacity:       aws.Int32(1),
			ClientToken:          aws.String(uuid.NewV4().String()),
			Type:                 types.FleetTypeRequest,
			IamFleetRole:         aws.String("arn:aws:iam::568126575653:role/aws-ec2-spot-fleet-tagging-role"),
			LaunchSpecifications: []types.SpotFleetLaunchSpecification{launchSpecification},
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

func (b *Bootstrapper) waitForInstance(ctx context.Context, name, requestID string, state types.InstanceStateName) (string, error) {
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

		for _, s := range out2.ActiveInstances {
			time.Sleep(time.Second * 1)
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

func (b *Bootstrapper) templateExists(ctx context.Context, name string) (bool, error) {
	t, err := handlers.GetLaunchTemplates(ctx, b.ec2Client)
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

func (b *Bootstrapper) findHostAMI(ctx context.Context, architecture types.ArchitectureValues) (string, error) {
	var nextToken *string

	for {
		out, err := b.ssmClient.GetParametersByPath(ctx, &ssm.GetParametersByPathInput{
			Path:      aws.String(AMI_PATH),
			NextToken: nextToken,
		})
		if err != nil {
			return "", err
		}

		for _, p := range out.Parameters {
			if strings.Contains(*p.Name, fmt.Sprintf("%s%s", API_PARAMETER_PREFIX, architecture)) {
				return *p.Value, nil
			}
		}

		nextToken = out.NextToken
		if nextToken == nil {
			break
		}
	}

	return "", fmt.Errorf("no ami found for architecture %s", architecture)
}
