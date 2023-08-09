package helpers

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/felipemarinho97/dev-spaces/util"
	"github.com/felipemarinho97/invest-path/clients"
)

func CreateElasticIP(ctx context.Context, client clients.IEC2Client, name string) (*ec2.AllocateAddressOutput, error) {
	out, err := client.AllocateAddress(ctx, &ec2.AllocateAddressInput{
		Domain: types.DomainTypeVpc,
		TagSpecifications: []types.TagSpecification{
			{
				ResourceType: types.ResourceTypeElasticIp,
				Tags:         util.GenerateTags(name),
			},
		},
	})
	if err != nil {
		return nil, err
	}

	return out, nil
}

func AssociateElasticIP(ctx context.Context, client clients.IEC2Client, instanceID string, allocationID string) (*ec2.AssociateAddressOutput, error) {
	out, err := client.AssociateAddress(ctx, &ec2.AssociateAddressInput{
		AllocationId: aws.String(allocationID),
		InstanceId:   aws.String(instanceID),
	})
	if err != nil {
		return nil, err
	}

	return out, nil
}

func DisassociateElasticIP(ctx context.Context, client clients.IEC2Client, associationId string) (*ec2.DisassociateAddressOutput, error) {
	out, err := client.DisassociateAddress(ctx, &ec2.DisassociateAddressInput{
		AssociationId: aws.String(associationId),
	})
	if err != nil {
		return nil, err
	}

	return out, nil
}

func releaseElasticIP(ctx context.Context, client clients.IEC2Client, allocationID string) (*ec2.ReleaseAddressOutput, error) {
	out, err := client.ReleaseAddress(ctx, &ec2.ReleaseAddressInput{
		AllocationId: aws.String(allocationID),
	})
	if err != nil {
		return nil, err
	}

	return out, nil
}

func CleanElasticIPs(ctx context.Context, client clients.IEC2Client, name string) error {
	addresses, err := client.DescribeAddresses(ctx, &ec2.DescribeAddressesInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("tag:dev-spaces:name"),
				Values: []string{name},
			},
		},
	})
	if err != nil {
		return err
	}

	for _, address := range addresses.Addresses {
		_, err := releaseElasticIP(ctx, client, *address.AllocationId)
		if err != nil {
			return err
		}
	}

	return nil
}

func GetElasticIP(ctx context.Context, client clients.IEC2Client, name string) (*types.Address, error) {
	addresses, err := client.DescribeAddresses(ctx, &ec2.DescribeAddressesInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("tag:dev-spaces:name"),
				Values: []string{name},
			},
		},
	})
	if err != nil {
		return nil, err
	}

	if len(addresses.Addresses) == 0 {
		return nil, nil
	}

	return &addresses.Addresses[0], nil
}
