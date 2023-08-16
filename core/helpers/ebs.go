package helpers

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/felipemarinho97/dev-spaces/core/util"
	"github.com/felipemarinho97/invest-path/clients"
	uuid "github.com/satori/go.uuid"
)

func CreateEBSVolume(ctx context.Context, client clients.IEC2Client, name string, size int32, az string) (*ec2.CreateVolumeOutput, error) {
	out, err := client.CreateVolume(ctx, &ec2.CreateVolumeInput{
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

func AttachEBSVolume(ctx context.Context, client clients.IEC2Client, instanceID string, volumeID string) error {
	_, err := client.AttachVolume(ctx, &ec2.AttachVolumeInput{
		Device:     aws.String("/dev/sdf"),
		InstanceId: aws.String(instanceID),
		VolumeId:   aws.String(volumeID),
	})
	if err != nil {
		return err
	}

	return nil
}

func DetachEBSVolume(ctx context.Context, client clients.IEC2Client, volumeID string) (*ec2.DetachVolumeOutput, error) {
	out, err := client.DetachVolume(ctx, &ec2.DetachVolumeInput{
		VolumeId: aws.String(volumeID),
		Force:    aws.Bool(false),
	})
	if err != nil {
		return nil, err
	}

	return out, nil
}

func WaitUntilEBSUnattached(ctx context.Context, client clients.IEC2Client, volumeID string) error {
	for {
		isEBSAttached, err := IsEBSAttached(ctx, client, volumeID)
		if err != nil {
			return err
		}

		if !isEBSAttached {
			return nil
		}
		time.Sleep(1 * time.Second)
	}
}

func IsEBSAttached(ctx context.Context, client clients.IEC2Client, volumeID string) (bool, error) {
	vol, err := client.DescribeVolumes(ctx, &ec2.DescribeVolumesInput{
		VolumeIds: []string{volumeID},
	})
	if err != nil {
		return false, err
	}
	if len(vol.Volumes) == 0 {
		return false, fmt.Errorf("no volume found with ID %s", volumeID)
	}
	if len(vol.Volumes[0].Attachments) == 0 || vol.Volumes[0].Attachments == nil {
		return false, nil
	}
	return true, nil
}

func WaitForEBSVolume(ctx context.Context, client clients.IEC2Client, volumeID string, state types.VolumeState) error {
	for {
		vol, err := client.DescribeVolumes(ctx, &ec2.DescribeVolumesInput{
			VolumeIds: []string{volumeID},
		})
		if err != nil {
			return err
		}
		if len(vol.Volumes) == 0 {
			return fmt.Errorf("no volume found with ID %s", volumeID)
		}
		if vol.Volumes[0].State == state {
			return nil
		}
		time.Sleep(1 * time.Second)
	}
}
