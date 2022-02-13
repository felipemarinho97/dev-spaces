package util

import "github.com/aws/aws-sdk-go-v2/service/ec2/types"

func IsManaged(tags []types.Tag) bool {
	for _, tag := range tags {
		if *tag.Key == "managed-by" && *tag.Value == "dev-spaces" {
			return true
		}
	}

	return false
}

func IsDevSpace(tags []types.Tag, devSpaceName string) bool {
	if devSpaceName == "" {
		return true
	}

	for _, tag := range tags {
		if *tag.Key == "dev-spaces:name" && *tag.Value == devSpaceName {
			return true
		}
	}

	return false
}
