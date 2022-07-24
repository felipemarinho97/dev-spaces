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

type AMIFilter struct {
	ID    string `validate:"required_without=Name"`
	Name  string `validate:"required_without=ID"`
	Arch  string
	Owner string
}

func GetImageFromFilter(ctx context.Context, client clients.IEC2Client, filter AMIFilter) (*types.Image, error) {
	input := &ec2.DescribeImagesInput{
		Filters: []types.Filter{},
		Owners:  []string{},
	}

	if filter.ID != "" {
		input.ImageIds = []string{filter.ID}
	}

	if filter.Name != "" {
		input.Filters = append(input.Filters, types.Filter{
			Name:   aws.String("name"),
			Values: []string{filter.Name},
		})
	}

	if filter.Arch != "" {
		input.Filters = append(input.Filters, types.Filter{
			Name:   aws.String("architecture"),
			Values: []string{filter.Arch},
		})
	}

	if filter.Owner != "" {
		input.Owners = append(input.Owners, filter.Owner)
	}

	out, err := client.DescribeImages(ctx, input)
	if err != nil {
		return nil, err
	}

	if len(out.Images) == 0 {
		return nil, fmt.Errorf("no image found with ID %s", filter.ID)
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

	return &out.Images[0], nil
}

// FindHostAMI returns the AMI ID for the host machine
func FindHostAMI(ctx context.Context, client clients.IEC2Client, architecture types.ArchitectureValues) (*types.Image, error) {
	return GetImageFromFilter(ctx, client, AMIFilter{
		Name:  "al2022-ami-minimal-*",
		Arch:  fmt.Sprintf("%s", architecture),
		Owner: "amazon",
	})
}
