package util

import "github.com/aws/aws-sdk-go-v2/service/ec2/types"

func IsManaged(tags []types.Tag) bool {
	for _, tag := range tags {
		if *tag.Key == "managed-by" && *tag.Value == "dev-space" {
			return true
		}
	}

	return false
}
