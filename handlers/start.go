package handlers

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/felipemarinho97/dev-spaces/util"
	awsUtil "github.com/felipemarinho97/invest-path/util"
	uuid "github.com/satori/go.uuid"
	"github.com/urfave/cli/v2"
)

func Create(c *cli.Context) error {
	ctx := c.Context
	memorySpec := c.Float64("min-memory")
	cpusSpec := c.Int("min-cpus")
	maxPrice := c.String("max-price")
	name := c.String("name")
	if name == "" {
		name = uuid.NewV4().String()
	}
	tName, tVersion := util.GetTemplateNameAndVersion(name)
	timeout := c.Duration("timeout")
	now := time.Now().UTC()
	now = time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), now.Second(), 0, time.UTC)

	config, err := awsUtil.LoadAWSConfig()
	config.Region = c.String("region")
	if err != nil {
		return err
	}

	client := ec2.NewFromConfig(config)

	minMemory := aws.Int32(int32(float64(1024) * memorySpec))

	template, err := getLaunchTemplateByName(ctx, client, tName)
	if err != nil {
		return err
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
						LaunchTemplateName: &tName,
						Version:            &tVersion,
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
									Min: minMemory,
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

	if err != nil {
		fmt.Println(err.Error())
		return err
	}

	id := out.SpotFleetRequestId
	fmt.Printf("spot-request-id=%v\n", *id)

	ub := util.NewUnknownBar("Waiting for instance request to be fulfilled...")
	ub.Start()

	for {
		time.Sleep(time.Second * 1)
		out2, err := client.DescribeSpotFleetInstances(ctx, &ec2.DescribeSpotFleetInstancesInput{
			SpotFleetRequestId: id,
		})
		if err != nil {
			ub.Stop()
			fmt.Println(err)
			return err
		}

		if len(out2.ActiveInstances) > 0 {
			instance := out2.ActiveInstances[0]
			ub.Stop()
			fmt.Printf("instance-id=%v\n", *instance.InstanceId)
			fmt.Printf("instance-type=%v\n", *instance.InstanceType)
			break
		}

	}
	return nil
}
