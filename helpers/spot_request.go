package helpers

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/felipemarinho97/dev-spaces/log"
	"github.com/felipemarinho97/dev-spaces/util"
	"github.com/felipemarinho97/invest-path/clients"
	"github.com/samber/lo"
	uuid "github.com/satori/go.uuid"
)

func CreateSpotRequest(ctx context.Context, client clients.IEC2Client, name, version string, cpusSpec, minMemory int, maxPrice string, template *types.LaunchTemplate, timeout time.Duration) (*ec2.RequestSpotFleetOutput, error) {
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
			IamFleetRole:                     aws.String("arn:aws:iam::568126575653:role/aws-ec2-spot-fleet-tagging-role"), // TODO: get this from config
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

type PreferedLaunchSpecs struct {
	InstanceType string
	MinMemory    int32
	MinCPU       int32
}

type CreateSpotTaskInput struct {
	Name                *string `validate:"required"`
	DeviceName          *string `validate:"required"`
	StorageSize         *int32  `validate:"required"`
	AMIID               *string `validate:"required"`
	KeyName             *string `validate:"required"`
	PreferedLaunchSpecs *PreferedLaunchSpecs
	InstanceProfileArn  *string
	StartupScript       *string
	Zone                *string
}

func CreateSpotTaskRunner(ctx context.Context, client clients.IEC2Client, in CreateSpotTaskInput) (*ec2.RequestSpotFleetOutput, error) {
	err := util.Validator.Struct(in)
	if err != nil {
		return nil, err
	}

	launchSpecification := types.SpotFleetLaunchSpecification{
		ImageId: in.AMIID,
		KeyName: in.KeyName,
		BlockDeviceMappings: []types.BlockDeviceMapping{
			{
				DeviceName: in.DeviceName,
				Ebs: &types.EbsBlockDevice{
					DeleteOnTermination: aws.Bool(false),
					Encrypted:           aws.Bool(true),
					Iops:                aws.Int32(3000),
					Throughput:          aws.Int32(125),
					VolumeSize:          in.StorageSize,
					VolumeType:          types.VolumeTypeGp3,
				},
			},
		},
		TagSpecifications: []types.SpotFleetTagSpecification{
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
			launchSpecification.InstanceRequirements = &types.InstanceRequirements{
				VCpuCount: &types.VCpuCountRange{
					Min: aws.Int32(in.PreferedLaunchSpecs.MinCPU),
				},
				MemoryMiB: &types.MemoryMiB{
					Min: aws.Int32(in.PreferedLaunchSpecs.MinMemory),
				},
				BareMetal:            types.BareMetalIncluded,
				BurstablePerformance: types.BurstablePerformanceIncluded,
			}
		}

	}

	if in.InstanceProfileArn != nil && *in.InstanceProfileArn != "" {
		launchSpecification.IamInstanceProfile = &types.IamInstanceProfileSpecification{
			Arn: in.InstanceProfileArn,
		}
	}

	if in.StartupScript != nil && *in.StartupScript != "" {
		encoded := base64.StdEncoding.EncodeToString([]byte(*in.StartupScript))
		launchSpecification.UserData = aws.String(encoded)
	}

	if in.Zone != nil && *in.Zone != "" {
		launchSpecification.Placement = &types.SpotPlacement{
			AvailabilityZone: in.Zone,
		}
	}

	input := &ec2.RequestSpotFleetInput{
		SpotFleetRequestConfig: &types.SpotFleetRequestConfigData{
			TargetCapacity:       aws.Int32(1),
			ClientToken:          aws.String(uuid.NewV4().String()),
			Type:                 types.FleetTypeRequest,
			IamFleetRole:         aws.String("arn:aws:iam::568126575653:role/aws-ec2-spot-fleet-tagging-role"),
			LaunchSpecifications: []types.SpotFleetLaunchSpecification{launchSpecification},
			TagSpecifications: []types.TagSpecification{
				{
					ResourceType: types.ResourceTypeSpotFleetRequest,
					Tags:         util.GenerateTags(*in.Name),
				},
			},
			AllocationStrategy: types.AllocationStrategyLowestPrice,
		},
	}

	out, err := client.RequestSpotFleet(ctx, input)
	if err != nil {
		fmt.Println(err.Error())
		return nil, err
	}

	return out, nil
}

func GetSpotRequestStatus(ctx context.Context, client clients.IEC2Client, name string) ([]types.SpotFleetRequestConfig, error) {
	requests, err := client.DescribeSpotFleetRequests(ctx, &ec2.DescribeSpotFleetRequestsInput{})
	if err != nil {
		fmt.Println(err.Error())
		return nil, err
	}

	var filteredRequests []types.SpotFleetRequestConfig
	for _, request := range requests.SpotFleetRequestConfigs {
		if util.IsManaged(request.Tags) && util.IsDevSpace(request.Tags, name) {
			filteredRequests = append(filteredRequests, request)
		}
	}

	return filteredRequests, nil
}

func GetCurrentSpotRequest(ctx context.Context, client clients.IEC2Client, name string) (*types.SpotFleetRequestConfig, error) {
	requests, err := GetSpotRequestStatus(ctx, client, name)
	if err != nil {
		return nil, err
	}
	if len(requests) == 0 {
		return nil, fmt.Errorf("no spot instance request found for %s", name)
	}

	f := lo.Filter(requests, func(sfrc types.SpotFleetRequestConfig, i int) bool {
		return sfrc.SpotFleetRequestState == types.BatchStateActive && sfrc.ActivityStatus == types.ActivityStatusFulfilled
	})

	if len(f) == 0 {
		return nil, fmt.Errorf("no active spot instance request found for %s", name)
	}

	current := f[0]

	return &current, nil
}

func CancelSpotRequest(ctx context.Context, client clients.IEC2Client, log log.Logger, name string) error {
	requests, err := client.DescribeSpotFleetRequests(ctx, &ec2.DescribeSpotFleetRequestsInput{})
	if err != nil {
		log.Error("Error getting spot requests:", err)
		return err
	}

	var requestID []string
	for _, request := range requests.SpotFleetRequestConfigs {
		if (request.SpotFleetRequestState == types.BatchStateActive || request.SpotFleetRequestState == types.BatchStateSubmitted) &&
			util.IsDevSpace(request.Tags, name) {
			requestID = append(requestID, *request.SpotFleetRequestId)
			break
		}
	}

	if len(requestID) == 0 {
		log.Error("No spot requests found")
		return nil
	}

	_, err = client.CancelSpotFleetRequests(ctx, &ec2.CancelSpotFleetRequestsInput{
		SpotFleetRequestIds: requestID,
		TerminateInstances:  aws.Bool(true),
	})
	if err != nil {
		log.Error("Error canceling spot request: ", err)
		return err
	}

	log.Info(fmt.Sprintf("Cancelled %d spot requests", len(requestID)))
	return nil
}
