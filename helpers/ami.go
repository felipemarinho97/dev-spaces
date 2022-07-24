package helpers

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/felipemarinho97/invest-path/clients"
)

// FindHostAMI returns the AMI ID for the host machine
func FindHostAMI(ctx context.Context, client clients.IEC2Client, architecture types.ArchitectureValues) (string, error) {
	out, err := client.DescribeImages(ctx, &ec2.DescribeImagesInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("name"),
				Values: []string{"al2022-ami-minimal-*"},
			},
			{
				Name:   aws.String("architecture"),
				Values: []string{fmt.Sprintf("%s", architecture)},
			},
		},
		Owners: []string{"amazon"},
	})
	if err != nil {
		return "", err
	}

	// sort by most recent
	sort.Slice(out.Images, func(i, j int) bool {
		dateI, err := time.Parse(time.RFC3339, *out.Images[i].CreationDate)
		if err != nil {
			return false
		}

		dateJ, err := time.Parse(time.RFC3339, *out.Images[j].CreationDate)
		if err != nil {
			return false
		}

		return dateI.After(dateJ)
	})

	if len(out.Images) == 0 {
		return "", fmt.Errorf("no ami found for architecture %s", architecture)
	}

	return *out.Images[0].ImageId, nil
}

func GetImage(ctx context.Context, client clients.IEC2Client, imageID string) (*types.Image, error) {
	images, err := client.DescribeImages(ctx, &ec2.DescribeImagesInput{
		ImageIds: []string{imageID},
	})
	if err != nil {
		return nil, err
	}

	if len(images.Images) == 0 {
		return nil, fmt.Errorf("no image found with ID %s", imageID)
	}

	return &images.Images[0], nil
}
