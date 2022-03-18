package handlers

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	awsUtil "github.com/felipemarinho97/invest-path/util"
	uuid "github.com/satori/go.uuid"
	"github.com/urfave/cli/v2"
)

func Create(c *cli.Context) error {
	ctx := c.Context
	name := c.String("name")

	config, err := awsUtil.LoadAWSConfig()
	config.Region = "us-east-1"
	if err != nil {
		return err
	}

	client := ec2.NewFromConfig(config)

	client.CreateLaunchTemplate(ctx, &ec2.CreateLaunchTemplateInput{
		LaunchTemplateName: &name,
		ClientToken:        aws.String(uuid.NewV4().String()),
		TagSpecifications: []types.TagSpecification{
			{
				ResourceType: types.ResourceTypeLaunchTemplate,
				Tags: []types.Tag{
					{
						Key:   aws.String("managed-by"),
						Value: aws.String("dev-spaces"),
					},
					{
						Key:   aws.String("dev-spaces:name"),
						Value: &name,
					},
				},
			},
		},
		LaunchTemplateData: &types.RequestLaunchTemplateData{
			MetadataOptions: &types.LaunchTemplateInstanceMetadataOptionsRequest{
				HttpEndpoint: types.LaunchTemplateInstanceMetadataEndpointStateEnabled,
			},
			BlockDeviceMappings: []types.LaunchTemplateBlockDeviceMappingRequest{
				{
					DeviceName: aws.String("/dev/xvda"),
					Ebs: &types.LaunchTemplateEbsBlockDeviceRequest{
						DeleteOnTermination: aws.Bool(true),
						Encrypted:           aws.Bool(true),
						Iops:                aws.Int32(3000),
						Throughput:          aws.Int32(125),
						VolumeSize:          aws.Int32(8),
						VolumeType:          types.VolumeTypeGp3,
					},
				},
			},
			UserData: aws.String(""),
		},
	})

	return nil
}
