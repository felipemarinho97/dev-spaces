package core

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/felipemarinho97/dev-spaces/core/helpers"
	"github.com/felipemarinho97/dev-spaces/core/log"
	"github.com/felipemarinho97/dev-spaces/core/util"
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

type StartOutput struct {
	// InstanceID is the instance id
	InstanceID string
	// Type is the instance type
	Type string
	// PublicIP is the public PublicIP of the instance
	PublicIP string
	// DNS is the DNS name of the instance
	DNS string
	// Port is the port to connect to the instance
	Port int
}

func (h *Handler) Start(ctx context.Context, startOptions StartOptions) (StartOutput, error) {
	log := h.Logger

	err := util.Validator.Struct(startOptions)
	if err != nil {
		return StartOutput{}, err
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
		return StartOutput{}, err
	}

	// get volume id from template tags
	volumeID := util.GetTag(template.Tags, "dev-spaces:volume-id")

	// wait until ebs volume is detached
	log.Info("Waiting for EBS volume to be available...")
	err = helpers.WaitUntilEBSUnattached(ctx, client, volumeID)
	if err != nil {
		return StartOutput{}, err
	}

	out, err := helpers.CreateSpotRequest(ctx, client, tName, tVersion, cpusSpec, minMemory, maxPrice, template, timeout)
	if err != nil {
		return StartOutput{}, err
	}

	fleetRequestID := out.FleetId
	log.Debug("Created spot fleet request with id: ", *fleetRequestID)

	// wait for instance to be running
	log.Info("Waiting for instance to be running...")
	instance, err := waitInstance(ctx, client, log, fleetRequestID)
	if err != nil {
		return StartOutput{}, err
	}

	ip := *instance.PublicIpAddress

	// attach ebs volume
	err = helpers.AttachEBSVolume(ctx, client, *instance.InstanceId, volumeID)
	if err != nil {
		return StartOutput{}, err
	}
	log.Info("Attached EBS volume with id: ", volumeID)

	return StartOutput{
		InstanceID: *instance.InstanceId,
		Type:       string(instance.InstanceType),
		PublicIP:   ip,
		Port:       2222,
		DNS:        *instance.PublicDnsName,
	}, nil
}

func waitInstance(ctx context.Context, client clients.IEC2Client, log log.Logger, id *string) (*types.Instance, error) {
	for {
		time.Sleep(time.Second * 1)
		out2, err := client.DescribeFleetInstances(ctx, &ec2.DescribeFleetInstancesInput{
			FleetId: id,
		})
		if err != nil {
			log.Error(err)
			return nil, err
		}

		if len(out2.ActiveInstances) > 0 {
			instance := out2.ActiveInstances[0]
			log.Info(fmt.Sprintf("Instance started with id: %s and type: %s", *instance.InstanceId, *instance.InstanceType))

			for {
				time.Sleep(time.Second * 1)
				out3, err := client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
					InstanceIds: []string{*instance.InstanceId},
				})
				if err != nil {
					log.Error(err)
					return nil, err
				}
				if len(out3.Reservations) > 0 {
					reservation := out3.Reservations[0]
					if len(reservation.Instances) > 0 {
						instance := reservation.Instances[0]
						if instance.State.Name == types.InstanceStateNameRunning {
							return &instance, nil
						}
					}
				}
			}
		}

	}
}
