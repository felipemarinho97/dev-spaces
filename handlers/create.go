package handlers

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/felipemarinho97/dev-spaces/helpers"
	"github.com/felipemarinho97/dev-spaces/util"
	"github.com/felipemarinho97/invest-path/clients"
	"gopkg.in/validator.v2"
)

type createTemplate struct {
	TemplateName         string             `validate:"nonzero,min=3,max=128"`
	DevSpaceAMIID        string             `validate:"nonzero"`
	PreferedInstanceType types.InstanceType `validate:"nonzero"`
	KeyName              string             `validate:"nonzero"`
	InstanceProfileArn   string
	StartupScript        string
	SecurityGroupIds     []string
	StorageSize          int32
	RootDeviceName       string
	HostStorageSize      int32
	HostAMIID            string
	HostArchitecture     types.ArchitectureValues
	HostAMI              *types.Image
	DevSpaceAMI          *types.Image
}

type CreateOptions struct {
	Name                 string       `validate:"nonzero,min=3,max=128"`
	DevSpaceAMIID        string       `validate:"nonzero"`
	PreferedInstanceType InstanceType `validate:"nonzero"`
	KeyName              string       `validate:"nonzero"`
	InstanceProfileArn   string
	StartupScriptPath    string
	SecurityGroupIds     []string
	StorageSize          int
	HostAMIID            string
}

type InstanceType types.InstanceType

type bootstrapper struct {
	ec2Client clients.IEC2Client
	// template  *createTemplate
	ub *util.UnknownBar
}

func (h *Handler) Create(ctx context.Context, opts CreateOptions) error {
	err := validator.Validate(opts)
	if err != nil {
		return err
	}
	name := opts.Name
	keyName := opts.KeyName
	instanceProfileArn := opts.InstanceProfileArn
	devSpaceAMIID := opts.DevSpaceAMIID
	hostAMIID := opts.HostAMIID
	startupScript := DEFAULT_STARTUP_SCRIPT
	preferedInstanceType := opts.PreferedInstanceType
	securityGroupIds := opts.SecurityGroupIds
	storageSize := int32(opts.StorageSize)
	ub := util.NewUnknownBar("Bootstrapping..")
	ub.Start()
	defer ub.Stop()

	client := h.EC2Client
	log := h.Logger

	// check if a launch template with the same name already exists
	templateExists, err := helpers.TemplateExists(ctx, client, name)
	if err != nil {
		return err
	}
	if templateExists {
		return fmt.Errorf("launch template with name %s already exists", name)
	}

	if opts.StartupScriptPath == "" {
		startupScript = DEFAULT_STARTUP_SCRIPT
		log.Info("Using default startup script...")
	} else {
		log.Info(fmt.Sprintf("Using custom startup script: %s", opts.StartupScriptPath))
		script, err := util.Readfile(opts.StartupScriptPath)
		if err != nil {
			return err
		}

		startupScript = script
	}

	// get the image of the dev space machine
	devSpaceAMI, err := helpers.GetImage(ctx, client, devSpaceAMIID)
	if err != nil {
		return fmt.Errorf("error describing host ami: %v", err)
	}

	if storageSize < *devSpaceAMI.BlockDeviceMappings[0].Ebs.VolumeSize {
		storageSize = *devSpaceAMI.BlockDeviceMappings[0].Ebs.VolumeSize
	}

	taskRunner, err := helpers.CreateSpotTaskRunner(ctx, client, helpers.CreateSpotTaskInput{
		Name:               &name,
		AMIID:              &devSpaceAMIID,
		DeviceName:         devSpaceAMI.RootDeviceName,
		StorageSize:        &storageSize,
		InstanceType:       aws.String(string(preferedInstanceType)),
		KeyName:            &keyName,
		InstanceProfileArn: &instanceProfileArn,
	})
	if err != nil {
		return err
	}
	log.Info(fmt.Sprintf("Spot task created: %s - Waiting instance to be assigned..", *taskRunner.SpotFleetRequestId))
	id, err := helpers.WaitForSpotFleetInstance(ctx, client, *taskRunner.SpotFleetRequestId, types.InstanceStateNameRunning)
	log.Info(fmt.Sprintf("Instance assigned: %s", id))
	if err != nil {
		return err
	}

	// get the volume id associated with the instance
	instanceData, err := helpers.GetInstanceData(ctx, client, id)
	if err != nil {
		return err
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
		return err
	}

	// cancel the spot task
	log.Info(fmt.Sprintf("Stopping instance: %s", id))
	_, err = client.CancelSpotFleetRequests(ctx, &ec2.CancelSpotFleetRequestsInput{
		SpotFleetRequestIds: []string{*taskRunner.SpotFleetRequestId},
		TerminateInstances:  aws.Bool(true),
	})
	if err != nil {
		return err
	}

	log.Info(fmt.Sprintf("Waiting for instance: %s to finish.. This may take a few minutes..", id))
	id, err = helpers.WaitForSpotFleetInstance(ctx, client, *taskRunner.SpotFleetRequestId, types.InstanceStateNameTerminated)
	if err != nil {
		return err
	}

	// get the best AMI to use for the devspace host
	if opts.HostAMIID == "" {
		hostAMI, err := helpers.FindHostAMI(ctx, client, devSpaceAMI.Architecture)
		if err != nil {
			return err
		}
		hostAMIID = hostAMI
	}

	// get the root device name fot this hostImage
	hostAMI, err := helpers.GetImage(ctx, client, hostAMIID)
	if err != nil {
		return fmt.Errorf("error retrieving host ami: %v", err)
	}
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
			AMIID: hostAMIID,
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
		return err
	}
	log.Info(fmt.Sprintf("Launch template created: %s", *o.LaunchTemplateId))

	log.Info(fmt.Sprintf("DevSpace \"%s\" created successfully.", name))

	return nil
}
