package handlers

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/felipemarinho97/dev-spaces/util"
	awsUtil "github.com/felipemarinho97/invest-path/util"
	"github.com/urfave/cli/v2"
)

func Stop(c *cli.Context) error {
	ctx := c.Context
	name := c.String("name")
	ub := util.NewUnknownBar("Stopping...")
	ub.Start()

	config, err := awsUtil.LoadAWSConfig()
	config.Region = c.String("region")
	if err != nil {
		return err
	}

	client := ec2.NewFromConfig(config)

	err = cancelSpotRequest(ctx, client, name, ub)
	if err != nil {
		return err
	}
	return nil
}

func cancelSpotRequest(ctx context.Context, client *ec2.Client, name string, ub *util.UnknownBar) error {
	requests, err := client.DescribeSpotFleetRequests(ctx, &ec2.DescribeSpotFleetRequestsInput{})
	if err != nil {
		fmt.Println(err)
		return err
	}

	var requestID []string
	for _, request := range requests.SpotFleetRequestConfigs {
		if (request.SpotFleetRequestState == types.BatchStateActive ||
			request.SpotFleetRequestState == types.BatchStateSubmitted) &&
			util.IsManaged(request.Tags) && util.IsDevSpace(request.Tags, name) {
			requestID = append(requestID, *request.SpotFleetRequestId)
			break
		}
	}

	if len(requestID) == 0 {
		ub.Stop()
		fmt.Println("OK")
		return nil
	}

	_, err = client.CancelSpotFleetRequests(ctx, &ec2.CancelSpotFleetRequestsInput{
		SpotFleetRequestIds: requestID,
		TerminateInstances:  aws.Bool(true),
	})
	if err != nil {
		ub.Stop()
		fmt.Println(err)
		return err
	}

	ub.Stop()
	fmt.Println("OK")
	return nil
}
