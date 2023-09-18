package helpers

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/felipemarinho97/invest-path/clients"
)

func SetTag(ctx context.Context, client clients.IEC2Client, resourceID, tagName, tagValue string) error {
	_, err := client.CreateTags(ctx, &ec2.CreateTagsInput{
		Resources: []string{
			resourceID,
		},
		Tags: []types.Tag{
			{
				Key:   aws.String(fmt.Sprintf("dev-spaces:%s", tagName)),
				Value: aws.String(tagValue),
			},
		},
	})
	if err != nil {
		return err
	}

	return nil
}

func GetTag(ctx context.Context, client clients.IEC2Client, resourceID, tagName string) (string, error) {
	tag, err := client.DescribeTags(ctx, &ec2.DescribeTagsInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("resource-id"),
				Values: []string{resourceID},
			},
			{
				Name:   aws.String("key"),
				Values: []string{fmt.Sprintf("dev-spaces:%s", tagName)},
			},
		},
	})
	if err != nil {
		return "", err
	}

	if len(tag.Tags) == 0 {
		return "", fmt.Errorf("no tag found with name %s", tagName)
	}

	return *tag.Tags[0].Value, nil
}
