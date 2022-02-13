package handlers

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/felipemarinho97/invest-path/util"
	uuid "github.com/satori/go.uuid"
	"github.com/urfave/cli/v2"
)

func Create(c *cli.Context) error {
	memorySpec := c.Int("min-memory")
	cpusSpec := c.Int("min-cpus")

	config, err := util.LoadAWSConfig()
	config.Region = "us-east-1"
	if err != nil {
		return err
	}

	client := ec2.NewFromConfig(config)

	out, err := client.RequestSpotFleet(context.Background(), &ec2.RequestSpotFleetInput{
		SpotFleetRequestConfig: &types.SpotFleetRequestConfigData{
			TagSpecifications: []types.TagSpecification{
				{
					ResourceType: "spot-fleet-request",
					Tags: []types.Tag{
						{
							Key:   aws.String("managed-by"),
							Value: aws.String("dev-space"),
						},
					},
				},
			},
			TargetCapacity:                   aws.Int32(1),
			IamFleetRole:                     aws.String("arn:aws:iam::568126575653:role/aws-ec2-spot-fleet-tagging-role"),
			AllocationStrategy:               types.AllocationStrategyLowestPrice,
			ClientToken:                      aws.String(uuid.NewV4().String()),
			ExcessCapacityTerminationPolicy:  types.ExcessCapacityTerminationPolicyDefault,
			InstanceInterruptionBehavior:     types.InstanceInterruptionBehaviorTerminate,
			TerminateInstancesWithExpiration: aws.Bool(true),
			Type:                             types.FleetTypeRequest,
			LaunchTemplateConfigs: []types.LaunchTemplateConfig{
				{
					LaunchTemplateSpecification: &types.FleetLaunchTemplateSpecification{
						LaunchTemplateId: aws.String("lt-000fa06f877b3cc29"),
						Version:          aws.String("5"),
					},
					Overrides: []types.LaunchTemplateOverrides{
						{
							AvailabilityZone: aws.String("us-east-1d"),
							SpotPrice:        aws.String("0.08"),
							InstanceRequirements: &types.InstanceRequirements{
								VCpuCount: &types.VCpuCountRange{
									Min: aws.Int32(int32(cpusSpec)),
								},
								MemoryMiB: &types.MemoryMiB{
									Min: aws.Int32(1024 * int32(memorySpec)),
								},
								BareMetal:            types.BareMetalIncluded,
								BurstablePerformance: types.BurstablePerformanceExcluded,
							},
						},
					},
				},
			},
			SpotMaxTotalPrice: aws.String("0.08"),
		},
	})

	if err != nil {
		fmt.Println(err.Error())
		return err
	}

	id := out.SpotFleetRequestId
	fmt.Printf("spot-request-id=%v\n", *id)

	for {
		time.Sleep(time.Second * 1)
		out2, err := client.DescribeSpotFleetInstances(context.Background(), &ec2.DescribeSpotFleetInstancesInput{
			SpotFleetRequestId: id,
		})
		if err != nil {
			fmt.Println(err)
			return err
		}

		if len(out2.ActiveInstances) > 0 {
			instance := out2.ActiveInstances[0]
			fmt.Printf("instance-id=%v\n", *instance.InstanceId)
			fmt.Printf("instance-type=%v\n", *instance.InstanceType)
			break
		}

	}
	return nil
}
