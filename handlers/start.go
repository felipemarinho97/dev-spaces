package handlers

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/felipemarinho97/dev-spaces/helpers"
	"github.com/felipemarinho97/dev-spaces/log"
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
	// Wait is a flag to wait for the instance to be running
	Wait bool
}

func (h Handler) Start(ctx context.Context, startOptions StartOptions) error {
	log := h.Logger
	ub := util.NewUnknownBar("Starting..")
	ub.Start()
	defer ub.Stop()

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
	wait := startOptions.Wait
	tName, tVersion := util.GetTemplateNameAndVersion(name)

	template, err := helpers.GetLaunchTemplateByName(ctx, client, tName)
	if err != nil {
		return err
	}

	out, err := helpers.CreateSpotRequest(ctx, client, tName, tVersion, cpusSpec, minMemory, maxPrice, template, timeout)
	if err != nil {
		return err
	}

	id := out.FleetId
	log.Debug("Created spot fleet request with id: ", *id)

	// wait for instance to be running
	log.Info("Waiting for instance to be running...")
	instanceID, err := waitInstance(ctx, client, log, id)
	if err != nil {
		return err
	}

	// create elastic ip
	eip, err := helpers.CreateElasticIP(ctx, client, name)
	if err != nil {
		return err
	}
	log.Info("Allocated elastic ip with address: ", *eip.PublicIp)

	// attach ebs volume
	volumeID := util.GetTag(template.Tags, "dev-spaces:volume-id")
	err = helpers.AttachEBSVolume(ctx, client, instanceID, volumeID)
	if err != nil {
		return err
	}
	log.Info("Attached EBS volume with id: ", volumeID)

	// associate elastic ip
	_, err = helpers.AssociateElasticIP(ctx, client, instanceID, *eip.AllocationId)
	if err != nil {
		return err
	}

	if wait {
		// wait until port 2222 is reachable
		log.Info("Waiting for port 2222 to be reachable...")
		err = helpers.WaitUntilReachable(*eip.PublicIp, 2222)
		if err != nil {
			return err
		}

		log.Info("You can now ssh into your dev space with the following command: ")
		fmt.Printf("$ ssh -i <your-key.pem> -p 2222 root@%s\n", *eip.PublicIp)
	}

	return nil
}

func waitInstance(ctx context.Context, client clients.IEC2Client, log log.Logger, id *string) (string, error) {
	for {
		time.Sleep(time.Second * 1)
		out2, err := client.DescribeFleetInstances(ctx, &ec2.DescribeFleetInstancesInput{
			FleetId: id,
		})
		if err != nil {
			log.Error(err)
			return "", err
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
