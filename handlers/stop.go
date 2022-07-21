package handlers

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/felipemarinho97/dev-spaces/util"
	"github.com/felipemarinho97/invest-path/clients"
)

type StopOptions struct {
	// Name of the dev space
	Name string
}

func (h *Handler) Stop(ctx context.Context, opts StopOptions) error {
	name := opts.Name
	ub := util.NewUnknownBar("Stopping...")
	ub.Start()
	defer ub.Stop()

	client := h.EC2Client

	err := cancelSpotRequest(ctx, client, name, ub)
	if err != nil {
		return err
	}
	fmt.Println("OK")
	return nil
}

func cancelSpotRequest(ctx context.Context, client clients.IEC2Client, name string, ub *util.UnknownBar) error {
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
		ub.SetDescription("No spot requests found")
		return nil
	}

	_, err = client.CancelSpotFleetRequests(ctx, &ec2.CancelSpotFleetRequestsInput{
		SpotFleetRequestIds: requestID,
		TerminateInstances:  aws.Bool(true),
	})
	if err != nil {
		fmt.Println(err)
		return err
	}

	ub.SetDescription(fmt.Sprintf("Cancelled %d spot requests", len(requestID)))
	return nil
}
