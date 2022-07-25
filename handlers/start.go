package handlers

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/felipemarinho97/dev-spaces/helpers"
	"github.com/felipemarinho97/dev-spaces/util"
	"github.com/felipemarinho97/invest-path/clients"
)

type StartOptions struct {
	// Name of the Dev Space
	Name string `validate:"required"`
	// MinMemory is the amount of memory in MiB
	MinMemory int `validate:"min=0"`
	// MinCPUs is the amount of cpus
	MinCPUs int `validate:"min=0"`
	// MaxPrice is the maximum price for the instance
	MaxPrice string `validate:"required"`
	// Timeout is the time in minutes to wait for the instance to be running
	Timeout time.Duration `validate:"min=0"`
}

func (h Handler) Start(ctx context.Context, startOptions StartOptions) error {
	err := util.Validator.Struct(startOptions)
	if err != nil {
		return err
	}

	client := h.EC2Client
	name := startOptions.Name
	minMemory := startOptions.MinMemory
	cpusSpec := startOptions.MinCPUs
	maxPrice := startOptions.MaxPrice
	timeout := startOptions.Timeout
	tName, tVersion := util.GetTemplateNameAndVersion(name)

	template, err := helpers.GetLaunchTemplateByName(ctx, client, tName)
	if err != nil {
		return err
	}

	out, err := helpers.CreateSpotRequest(ctx, client, tName, tVersion, cpusSpec, minMemory, maxPrice, template, timeout)
	if err != nil {
		return err
	}

	id := out.SpotFleetRequestId
	fmt.Printf("spot-request-id=%v\n", *id)

	ub := util.NewUnknownBar("Waiting for instance request to be fulfilled...")
	ub.Start()
	defer ub.Stop()

	// wait for instance to be running
	instanceID, err := waitInstance(client, ctx, id, ub)
	if err != nil {
		return err
	}

	// attach ebs volume
	volumeID := util.GetTag(template.Tags, "dev-spaces:volume-id")
	err = helpers.AttachEBSVolume(ctx, client, instanceID, volumeID)
	if err != nil {
		return err
	}

	return nil
}

func waitInstance(client clients.IEC2Client, ctx context.Context, id *string, ub *util.UnknownBar) (string, error) {
	for {
		time.Sleep(time.Second * 1)
		out2, err := client.DescribeSpotFleetInstances(ctx, &ec2.DescribeSpotFleetInstancesInput{
			SpotFleetRequestId: id,
		})
		if err != nil {
			ub.Stop()
			fmt.Println(err)
			return "", err
		}

		if len(out2.ActiveInstances) > 0 {
			instance := out2.ActiveInstances[0]
			ub.Stop()
			fmt.Printf("instance-id=%v\n", *instance.InstanceId)
			fmt.Printf("instance-type=%v\n", *instance.InstanceType)

			for {
				time.Sleep(time.Second * 1)
				out3, err := client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
					InstanceIds: []string{*instance.InstanceId},
				})
				if err != nil {
					fmt.Println(err)
					return "", err
				}
				if len(out3.Reservations) > 0 {
					reservation := out3.Reservations[0]
					if len(reservation.Instances) > 0 {
						instance := reservation.Instances[0]
						if instance.State.Name == types.InstanceStateNameRunning {
							return *instance.InstanceId, nil
						}
					}
				}
			}
		}

	}
}
