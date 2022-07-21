package handlers

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/felipemarinho97/dev-spaces/util"
	"github.com/felipemarinho97/invest-path/clients"
	uuid "github.com/satori/go.uuid"
	"gopkg.in/validator.v2"
)

type StartOptions struct {
	// Name of the Dev Space
	Name string `validate:"nonzero"`
	// MinMemory is the amount of memory in MiB
	MinMemory int `validate:"min=0"`
	// MinCPUs is the amount of cpus
	MinCPUs int `validate:"min=0"`
	// MaxPrice is the maximum price for the instance
	MaxPrice string `validate:"nonzero"`
	// Timeout is the time in minutes to wait for the instance to be running
	Timeout time.Duration `validate:"min=0"`
}

func (h Handler) Start(ctx context.Context, startOptions StartOptions) error {
	err := validator.Validate(startOptions)
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

	template, err := getLaunchTemplateByName(ctx, client, tName)
	if err != nil {
		return err
	}

	out, err := createSpotRequest(ctx, client, name, tVersion, cpusSpec, minMemory, maxPrice, template, timeout)
	if err != nil {
		return err
	}

	id := out.SpotFleetRequestId
	fmt.Printf("spot-request-id=%v\n", *id)

	ub := util.NewUnknownBar("Waiting for instance request to be fulfilled...")
	ub.Start()

	// wait for instance to be running
	instanceID, err := waitInstance(client, ctx, id, ub)
	if err != nil {
		return err
	}

	// attach ebs volume
	volumeID := util.GetTag(template.Tags, "dev-spaces:volume-id")
	err = attachEBSVolume(ctx, client, instanceID, volumeID)

	return nil
}

func createSpotRequest(ctx context.Context, client clients.IEC2Client, name, version string, cpusSpec, minMemory int, maxPrice string, template *types.LaunchTemplate, timeout time.Duration) (*ec2.RequestSpotFleetOutput, error) {
	now := time.Now().UTC()
	now = time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), now.Second(), 0, time.UTC)

	if version == "" {
		version = "$Default"
	}

	out, err := client.RequestSpotFleet(ctx, &ec2.RequestSpotFleetInput{
		SpotFleetRequestConfig: &types.SpotFleetRequestConfigData{
			TagSpecifications: []types.TagSpecification{
				{
					ResourceType: types.ResourceTypeSpotFleetRequest,
					Tags:         util.GenerateTags(name),
				},
			},
			TargetCapacity:                   aws.Int32(1),
			IamFleetRole:                     aws.String("arn:aws:iam::568126575653:role/aws-ec2-spot-fleet-tagging-role"),
			AllocationStrategy:               types.AllocationStrategyLowestPrice,
			ClientToken:                      aws.String(uuid.NewV4().String()),
			ExcessCapacityTerminationPolicy:  types.ExcessCapacityTerminationPolicyDefault,
			InstanceInterruptionBehavior:     types.InstanceInterruptionBehaviorTerminate,
			TerminateInstancesWithExpiration: aws.Bool(true),
			ValidFrom:                        aws.Time(now),
			ValidUntil:                       aws.Time(now.Add(timeout)),
			Type:                             types.FleetTypeRequest,
			LaunchTemplateConfigs: []types.LaunchTemplateConfig{
				{
					LaunchTemplateSpecification: &types.FleetLaunchTemplateSpecification{
						LaunchTemplateId: template.LaunchTemplateId,
						Version:          &version,
					},
					Overrides: []types.LaunchTemplateOverrides{
						{
							SpotPrice:        &maxPrice,
							AvailabilityZone: aws.String(util.GetTag(template.Tags, "dev-spaces:zone")),
							InstanceRequirements: &types.InstanceRequirements{
								VCpuCount: &types.VCpuCountRange{
									Min: aws.Int32(int32(cpusSpec)),
								},
								MemoryMiB: &types.MemoryMiB{
									Min: aws.Int32(int32(minMemory)),
								},
								BareMetal:            types.BareMetalIncluded,
								BurstablePerformance: types.BurstablePerformanceIncluded,
							},
						},
					},
				},
			},
			SpotMaxTotalPrice: &maxPrice,
		},
	})

	return out, err
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

func attachEBSVolume(ctx context.Context, client clients.IEC2Client, instanceID string, volumeID string) error {
	out, err := client.AttachVolume(ctx, &ec2.AttachVolumeInput{
		Device:     aws.String("/dev/sdf"),
		InstanceId: aws.String(instanceID),
		VolumeId:   aws.String(volumeID),
	})
	if err != nil {
		return err
	}

	fmt.Printf("ebs-volume-id=%v\n", *out.VolumeId)

	return nil
}
