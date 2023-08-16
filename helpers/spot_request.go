package helpers

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/felipemarinho97/dev-spaces/log"
	"github.com/felipemarinho97/dev-spaces/util"
	"github.com/felipemarinho97/invest-path/clients"
	"github.com/samber/lo"
	uuid "github.com/satori/go.uuid"
)

func CreateSpotRequest(ctx context.Context, client clients.IEC2Client, name, version string, cpusSpec, minMemory int, maxPrice string, template *types.LaunchTemplate, timeout time.Duration) (*ec2.CreateFleetOutput, error) {
	now := time.Now().UTC()
	now = time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), now.Second(), 0, time.UTC)

	if version == "" {
		version = "$Default"
	}

	out, err := client.CreateFleet(ctx, &ec2.CreateFleetInput{
		LaunchTemplateConfigs: []types.FleetLaunchTemplateConfigRequest{
			{
				LaunchTemplateSpecification: &types.FleetLaunchTemplateSpecificationRequest{
					LaunchTemplateId: template.LaunchTemplateId,
					Version:          &version,
				},
				Overrides: []types.FleetLaunchTemplateOverridesRequest{
					{
						AvailabilityZone: aws.String(util.GetTag(template.Tags, "dev-spaces:zone")),
						InstanceRequirements: &types.InstanceRequirementsRequest{
							VCpuCount: &types.VCpuCountRangeRequest{
								Min: aws.Int32(int32(cpusSpec)),
							},
							MemoryMiB: &types.MemoryMiBRequest{
								Min: aws.Int32(int32(minMemory)),
							},
							BareMetal:            types.BareMetalIncluded,
							BurstablePerformance: types.BurstablePerformanceIncluded,
						},
						MaxPrice: &maxPrice,
					},
				},
			},
		},
		TargetCapacitySpecification: &types.TargetCapacitySpecificationRequest{
			TotalTargetCapacity:       aws.Int32(1),
			DefaultTargetCapacityType: types.DefaultTargetCapacityTypeSpot,
			OnDemandTargetCapacity:    aws.Int32(0),
			SpotTargetCapacity:        aws.Int32(1),
		},
		ClientToken:                      aws.String(uuid.NewV4().String()),
		ExcessCapacityTerminationPolicy:  types.FleetExcessCapacityTerminationPolicyTermination,
		TerminateInstancesWithExpiration: aws.Bool(true),
		Type:                             types.FleetTypeRequest,
		ValidFrom:                        aws.Time(now),
		ValidUntil:                       aws.Time(now.Add(timeout)),
		SpotOptions: &types.SpotOptionsRequest{
			AllocationStrategy:           types.SpotAllocationStrategyLowestPrice,
			InstanceInterruptionBehavior: types.SpotInstanceInterruptionBehaviorTerminate,
			MaxTotalPrice:                &maxPrice,
		},
		TagSpecifications: []types.TagSpecification{

			{
				ResourceType: types.ResourceTypeFleet,
				Tags:         util.GenerateTags(name),
			},
		},
	})
	if err != nil {
		return nil, err
	}

	return out, err
}

type PreferedLaunchSpecs struct {
	InstanceType string
	MinMemory    int32
	MinCPU       int32
}

type CreateSpotTaskInput struct {
	Name                      *string `validate:"required"`
	DeviceName                *string `validate:"required"`
	StorageSize               *int32  `validate:"required"`
	AMIID                     *string `validate:"required"`
	KeyName                   *string `validate:"required"`
	PreferedLaunchSpecs       *PreferedLaunchSpecs
	InstanceProfileArn        *string
	StartupScript             *string
	Zone                      *string
	DeleteVolumeOnTermination bool
}

func CreateSpotTaskRunner(ctx context.Context, client clients.IEC2Client, in CreateSpotTaskInput) (*ec2.CreateFleetOutput, error) {
	err := util.Validator.Struct(in)
	if err != nil {
		return nil, err
	}

	launchSpecification := types.RequestLaunchTemplateData{
		ImageId: in.AMIID,
		KeyName: in.KeyName,
		BlockDeviceMappings: []types.LaunchTemplateBlockDeviceMappingRequest{
			{
				DeviceName: in.DeviceName,
				Ebs: &types.LaunchTemplateEbsBlockDeviceRequest{
					DeleteOnTermination: &in.DeleteVolumeOnTermination,
					Encrypted:           aws.Bool(true),
					Iops:                aws.Int32(3000),
					Throughput:          aws.Int32(125),
					VolumeSize:          in.StorageSize,
					VolumeType:          types.VolumeTypeGp3,
				},
			},
		},
		TagSpecifications: []types.LaunchTemplateTagSpecificationRequest{
			{
				ResourceType: types.ResourceTypeInstance,
				Tags:         util.GenerateTags(*in.Name),
			},
		},
	}

	if in.PreferedLaunchSpecs != nil {
		if in.PreferedLaunchSpecs.InstanceType != "" {
			launchSpecification.InstanceType = types.InstanceType(in.PreferedLaunchSpecs.InstanceType)
		} else {
			launchSpecification.InstanceRequirements = &types.InstanceRequirementsRequest{
				VCpuCount: &types.VCpuCountRangeRequest{
					Min: aws.Int32(in.PreferedLaunchSpecs.MinCPU),
				},
				MemoryMiB: &types.MemoryMiBRequest{
					Min: aws.Int32(in.PreferedLaunchSpecs.MinMemory),
				},
				BareMetal:            types.BareMetalIncluded,
				BurstablePerformance: types.BurstablePerformanceIncluded,
			}
		}

	}

	if in.InstanceProfileArn != nil && *in.InstanceProfileArn != "" {
		launchSpecification.IamInstanceProfile = &types.LaunchTemplateIamInstanceProfileSpecificationRequest{
			Arn: in.InstanceProfileArn,
		}
	}

	if in.StartupScript != nil && *in.StartupScript != "" {
		encoded := base64.StdEncoding.EncodeToString([]byte(*in.StartupScript))
		launchSpecification.UserData = aws.String(encoded)
	}

	if in.Zone != nil && *in.Zone != "" {
		launchSpecification.Placement = &types.LaunchTemplatePlacementRequest{
			AvailabilityZone: in.Zone,
		}
	}

	lt, err := client.CreateLaunchTemplate(ctx, &ec2.CreateLaunchTemplateInput{
		LaunchTemplateName: aws.String(fmt.Sprintf("%s-runner", *in.Name)),
		LaunchTemplateData: &launchSpecification,
	})
	if err != nil {
		return nil, err
	}

	input := &ec2.CreateFleetInput{
		LaunchTemplateConfigs: []types.FleetLaunchTemplateConfigRequest{
			{
				LaunchTemplateSpecification: &types.FleetLaunchTemplateSpecificationRequest{
					LaunchTemplateId: lt.LaunchTemplate.LaunchTemplateId,
					Version:          aws.String("$Latest"),
				},
			},
		},
		ClientToken: aws.String(uuid.NewV4().String()),
		Type:        types.FleetTypeRequest,
		TargetCapacitySpecification: &types.TargetCapacitySpecificationRequest{
			DefaultTargetCapacityType: types.DefaultTargetCapacityTypeSpot,
			TotalTargetCapacity:       aws.Int32(1),
			SpotTargetCapacity:        aws.Int32(1),
			OnDemandTargetCapacity:    aws.Int32(0),
		},
		SpotOptions: &types.SpotOptionsRequest{
			AllocationStrategy: types.SpotAllocationStrategyLowestPrice,
		},
		TagSpecifications: []types.TagSpecification{
			{
				ResourceType: types.ResourceTypeFleet,
				Tags:         util.GenerateTags(*in.Name),
			},
		},
	}
	out, err := client.CreateFleet(ctx, input)
	if err != nil {
		fmt.Println(err.Error())
		return nil, err
	}

	return out, nil
}

func GetFleetStatus(ctx context.Context, client clients.IEC2Client, name string) ([]types.FleetData, error) {
	requests, err := client.DescribeFleets(ctx, &ec2.DescribeFleetsInput{})
	if err != nil {
		fmt.Println(err.Error())
		return nil, err
	}

	var filteredRequests []types.FleetData
	for _, request := range requests.Fleets {
		if util.IsManaged(request.Tags) && util.IsDevSpace(request.Tags, name) {
			filteredRequests = append(filteredRequests, request)
		}
	}

	return filteredRequests, nil
}

func GetCurrentFleetRequest(ctx context.Context, client clients.IEC2Client, name string) (*types.FleetData, error) {
	requests, err := GetFleetStatus(ctx, client, name)
	if err != nil {
		return nil, err
	}
	if len(requests) == 0 {
		return nil, fmt.Errorf("no spot instance request found for %s", name)
	}

	f := lo.Filter(requests, func(sfrc types.FleetData, i int) bool {
		return sfrc.FleetState == types.FleetStateCodeActive && sfrc.ActivityStatus == types.FleetActivityStatusFulfilled
	})

	if len(f) == 0 {
		return nil, fmt.Errorf("no active spot instance request found for %s", name)
	}

	current := f[0]

	return &current, nil
}

func CancelSpotRequests(ctx context.Context, client clients.IEC2Client, log log.Logger, name string) (int, error) {
	requests, err := client.DescribeFleets(ctx, &ec2.DescribeFleetsInput{})
	if err != nil {
		log.Error("Error getting spot requests:", err)
		return 0, err
	}

	var requestID []string
	for _, request := range requests.Fleets {
		if (request.FleetState == types.FleetStateCodeActive || request.FleetState == types.FleetStateCodeSubmitted) &&
			util.IsDevSpace(request.Tags, name) {
			requestID = append(requestID, *request.FleetId)
		}
	}

	if len(requestID) == 0 {
		log.Error("No spot requests found")
		return 0, nil
	}

	err = CancelFleetRequests(ctx, client, requestID)
	if err != nil {
		log.Error("Error cancelling spot request:", err)
		return 0, err
	}

	log.Info(fmt.Sprintf("Cancelled %d spot requests", len(requestID)))
	return len(requestID), nil
}

func CancelFleetRequests(ctx context.Context, client clients.IEC2Client, ids []string) error {
	_, err := client.DeleteFleets(ctx, &ec2.DeleteFleetsInput{
		FleetIds:           ids,
		TerminateInstances: aws.Bool(true),
	})
	if err != nil {
		return err
	}

	return nil
}
