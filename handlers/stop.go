package handlers

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/felipemarinho97/dev-spaces/util"
	"github.com/urfave/cli/v2"
)

func Stop(c *cli.Context) error {
	config, err := util.LoadAWSConfig()
	config.Region = "us-east-1"
	if err != nil {
		return err
	}

	client := ec2.NewFromConfig(config)

	requests, err := client.DescribeSpotFleetRequests(context.Background(), &ec2.DescribeSpotFleetRequestsInput{})
	if err != nil {
		fmt.Println(err)
		return err
	}

	var requestID []string
	for _, request := range requests.SpotFleetRequestConfigs {
		if (request.SpotFleetRequestState == types.BatchStateActive ||
			request.SpotFleetRequestState == types.BatchStateSubmitted) &&
			util.IsManaged(request.Tags) {
			requestID = append(requestID, *request.SpotFleetRequestId)
			break
		}
	}

	_, err = client.CancelSpotFleetRequests(context.Background(), &ec2.CancelSpotFleetRequestsInput{
		SpotFleetRequestIds: requestID,
		TerminateInstances:  aws.Bool(true),
	})
	if err != nil {
		fmt.Println(err)
		return err
	}

	return nil
}
