package core

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/felipemarinho97/dev-spaces/core/helpers"
	"github.com/felipemarinho97/dev-spaces/core/util"
)

type PreferedLaunchSpecs struct {
	InstanceType InstanceType `validate:"required_without=MinMemory,required_without=MinCPU" yaml:"instance_type"`
	MinMemory    int32        `validate:"required_without=InstanceType" yaml:"min_memory"`
	MinCPU       int32        `validate:"required_without=InstanceType" yaml:"min_cpu"`
}

type AMIFilter struct {
	ID    string `validate:"required_without=Name" yaml:"id"`
	Name  string `validate:"required_without=ID" yaml:"name"`
	Arch  string `yaml:"arch"`
	Owner string `yaml:"owner"`
}

type CreateOptions struct {
	Name                string              `validate:"required,min=3,max=128"`
	DevSpaceAMI         AMIFilter           `validate:"required"`
	PreferedLaunchSpecs PreferedLaunchSpecs `validate:"required"`
	KeyName             string              `validate:"required"`
	InstanceProfileArn  string
	StartupScriptPath   string
	SecurityGroupIds    []string
	StorageSize         int
	HostAMI             *AMIFilter
}

type CreateOutput struct {
	LaunchTemplateId *string
	HostImage        *string
	DevSpaceImage    *string
	StorageVolumeId  *string
	StorageSize      *int
}

type InstanceType types.InstanceType

func (h *Handler) Create(ctx context.Context, opts CreateOptions) (CreateOutput, error) {
	err := util.Validator.Struct(opts)
	if err != nil {
		return CreateOutput{}, err
	}
	name := opts.Name
	keyName := opts.KeyName
	instanceProfileArn := opts.InstanceProfileArn
	startupScript := DEFAULT_STARTUP_SCRIPT
	preferedInstanceType := opts.PreferedLaunchSpecs.InstanceType
	securityGroupIds := opts.SecurityGroupIds
	storageSize := int32(opts.StorageSize)

	client := h.EC2Client
	log := h.Logger

	// check if a launch template with the same name already exists
	templateExists, err := helpers.TemplateExists(ctx, client, name)
	if err != nil {
		return CreateOutput{}, err
	}
	if templateExists {
		return CreateOutput{}, fmt.Errorf("launch template with name %s already exists", name)
	}

	if opts.StartupScriptPath == "" {
		startupScript = DEFAULT_STARTUP_SCRIPT
		log.Info("Using default startup script...")
	} else {
		log.Info(fmt.Sprintf("Using custom startup script: %s", opts.StartupScriptPath))
		script, err := util.RetrieveFile(opts.StartupScriptPath)
		if err != nil {
			return CreateOutput{}, err
		}

		startupScript = script
	}

	// validate key pair
	_, err = helpers.GetKeyPair(ctx, client, keyName)
	if err != nil {
		return CreateOutput{}, err
	}

	// get the image of the dev space machine
	devSpaceAMI, err := helpers.GetImageFromFilter(ctx, client, helpers.AMIFilter{
		ID:    opts.DevSpaceAMI.ID,
		Name:  opts.DevSpaceAMI.Name,
		Arch:  opts.DevSpaceAMI.Arch,
		Owner: opts.DevSpaceAMI.Owner,
	})
	if err != nil {
		return CreateOutput{}, fmt.Errorf("error describing host ami: %v", err)
	}
	log.Debug(fmt.Sprintf("AMI filter used: ID '%s', Name '%s', Arch '%s', Owner '%s'", opts.DevSpaceAMI.ID, opts.DevSpaceAMI.Name, opts.DevSpaceAMI.Arch, opts.DevSpaceAMI.Owner))
	log.Info(fmt.Sprintf("Using dev space AMI: %s", *devSpaceAMI.ImageLocation))

	// get the best AMI to use for the devspace host
	var hostAMI *types.Image
	if opts.HostAMI == nil {
		hostAMI, err = helpers.FindHostAMI(ctx, client, devSpaceAMI.Architecture)
		if err != nil {
			return CreateOutput{}, err
		}
	} else {
		hostAMI, err = helpers.GetImageFromFilter(ctx, client, helpers.AMIFilter{
			ID:    opts.HostAMI.ID,
			Name:  opts.HostAMI.Name,
			Arch:  opts.HostAMI.Arch,
			Owner: opts.HostAMI.Owner,
		})
		if err != nil {
			return CreateOutput{}, err
		}
	}
	log.Debug(fmt.Sprintf("Using host AMI: %s", *hostAMI.ImageLocation))

	if storageSize < *devSpaceAMI.BlockDeviceMappings[0].Ebs.VolumeSize {
		storageSize = *devSpaceAMI.BlockDeviceMappings[0].Ebs.VolumeSize
	}

	taskRunner, err := helpers.CreateSpotTaskRunner(ctx, client, helpers.CreateSpotTaskInput{
		Name:        &name,
		AMIID:       devSpaceAMI.ImageId,
		DeviceName:  devSpaceAMI.RootDeviceName,
		StorageSize: &storageSize,
		PreferedLaunchSpecs: &helpers.PreferedLaunchSpecs{
			InstanceType: string(preferedInstanceType),
			MinMemory:    opts.PreferedLaunchSpecs.MinMemory,
			MinCPU:       opts.PreferedLaunchSpecs.MinCPU,
		},
		KeyName:            &keyName,
		InstanceProfileArn: &instanceProfileArn,
	})
	if err != nil {
		return CreateOutput{}, err
	}
	log.Info(fmt.Sprintf("Spot task created: %s - Waiting instance to be assigned..", *taskRunner.FleetId))
	id, err := helpers.WaitForFleetInstance(ctx, client, *taskRunner.FleetId, types.InstanceStateNameRunning)
	if err != nil {
		return CreateOutput{}, err
	}
	log.Info(fmt.Sprintf("Instance assigned: %s", id))

	// get the volume id associated with the instance
	instanceData, err := helpers.GetInstanceData(ctx, client, id)
	if err != nil {
		return CreateOutput{}, err
	}
	volumeId := instanceData.BlockDeviceMappings[0].Ebs.VolumeId
	volumeZone := instanceData.Placement.AvailabilityZone

	// tag the volume
	log.Info(fmt.Sprintf("Tagging volume: %s", *volumeId))
	_, err = client.CreateTags(ctx, &ec2.CreateTagsInput{
		Resources: []string{*volumeId},
		Tags:      util.GenerateTags(name),
	})
	if err != nil {
		return CreateOutput{}, err
	}

	// cancel the spot task
	log.Info(fmt.Sprintf("Stopping instance: %s", id))
	err = helpers.CancelFleetRequests(ctx, client, []string{*taskRunner.FleetId})
	if err != nil {
		return CreateOutput{}, err
	}

	// delete the runner template
	err = helpers.DeleteLaunchTemplate(ctx, client, name+"-runner")
	if err != nil {
		return CreateOutput{}, err
	}

	log.Info(fmt.Sprintf("Waiting for instance: %s to finish.. This may take a few minutes..", id))
	id, err = helpers.WaitForFleetInstance(ctx, client, *taskRunner.FleetId, types.InstanceStateNameTerminated)
	if err != nil {
		return CreateOutput{}, err
	}
	// wait for ebs volume to be available
	helpers.WaitUntilEBSUnattached(ctx, client, *volumeId)

	// get the root device name fot this hostImage
	hostDeviceName := *hostAMI.RootDeviceName
	hostStorageSize := *hostAMI.BlockDeviceMappings[0].Ebs.VolumeSize

	// create the launch template
	o, err := helpers.CreateLaunchTemplate(ctx, client, log, helpers.CreateLaunchTemplateInput{
		Name:               name,
		VolumeId:           *volumeId,
		VolumeZone:         *volumeZone,
		StartupScript:      startupScript,
		SecurityGroupIds:   securityGroupIds,
		InstanceProfileArn: &instanceProfileArn,
		KeyName:            keyName,
		Host: helpers.CreateLaunchTemplateHost{
			AMIID: *hostAMI.ImageId,
			Device: helpers.CreateLaunchTemplateHostDevice{
				Name:       hostDeviceName,
				Size:       hostStorageSize,
				Type:       string(hostAMI.BlockDeviceMappings[0].Ebs.VolumeType),
				IOPS:       hostAMI.BlockDeviceMappings[0].Ebs.Iops,
				Throughput: hostAMI.BlockDeviceMappings[0].Ebs.Throughput,
			},
		},
	})
	if err != nil {
		return CreateOutput{}, err
	}
	log.Info(fmt.Sprintf("Launch template created: %s", *o.LaunchTemplateId))

	return CreateOutput{
		LaunchTemplateId: o.LaunchTemplateId,
		HostImage:        hostAMI.ImageLocation,
		DevSpaceImage:    devSpaceAMI.ImageLocation,
		StorageVolumeId:  volumeId,
		StorageSize:      aws.Int(int(storageSize)),
	}, nil
}
