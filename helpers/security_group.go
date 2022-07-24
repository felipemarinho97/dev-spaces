package helpers

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/felipemarinho97/dev-spaces/log"
	"github.com/felipemarinho97/dev-spaces/util"
	"github.com/felipemarinho97/invest-path/clients"
)

func CreateSecurityGroup(ctx context.Context, client clients.IEC2Client, log log.Logger, name string) (*string, error) {
	log.Info("Creating security group..")

	// get th default vpc id
	vpc, err := client.DescribeVpcs(ctx, &ec2.DescribeVpcsInput{
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
	out, err := client.CreateSecurityGroup(ctx, &ec2.CreateSecurityGroupInput{
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
	log.Info("Adding ingress rules for ssh..")
	_, err = client.AuthorizeSecurityGroupIngress(ctx, &ec2.AuthorizeSecurityGroupIngressInput{
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
