package handlers

import (
	"context"
	"fmt"
	"time"

	"github.com/felipemarinho97/dev-spaces/helpers"
	"github.com/felipemarinho97/dev-spaces/util"
	"github.com/felipemarinho97/dev-spaces/util/ssh"
)

type EditSpecOptions struct {
	// Name of the Dev Space
	Name string `validate:"required"`
	// MinMemory is the amount of memory in MiB
	MinMemory int `validate:"min=0"`
	// MinCPUs is the amount of cpus
	MinCPUs int `validate:"min=0"`
	// MaxPrice is the maximum price for the instance
	MaxPrice string `validate:"required"`
	// SSHKey is the path of the SSH key
	SSHKey string `validate:"required"`
}

type EditOutput struct {
	// InstanceID is the ID of the instance
	InstanceID string `json:"instance_id"`
	// InstanceIP is the Public IP of the instance
	InstanceIP string `json:"instance_ip"`
	// InstanceType is the type of the instance
	InstanceType string `json:"instance_type"`
	// FleetRequestID is the ID of the FleetRequest
	FleetRequestID string `json:"fleet_request_id"`
}

func (h *Handler) EditSpec(ctx context.Context, opts EditSpecOptions) (EditOutput, error) {
	log := h.Logger
	ub := util.NewUnknownBar("Editing..")
	ub.Start()
	defer ub.Stop()

	err := util.Validator.Struct(opts)
	if err != nil {
		return EditOutput{}, err
	}

	identityKey, err := util.Readfile(opts.SSHKey)
	if err != nil {
		return EditOutput{}, err
	}

	client := h.EC2Client

	name, version := util.GetTemplateNameAndVersion(opts.Name)
	template, err := helpers.GetLaunchTemplateByName(ctx, client, name)
	if err != nil {
		return EditOutput{}, err
	}
	volumeID := util.GetTag(template.Tags, "dev-spaces:volume-id")

	// get current spot instance request
	currentReq, err := helpers.GetCurrentFleetRequest(ctx, client, name)
	if err != nil {
		return EditOutput{}, err
	}
	currentInstance, err := waitInstance(ctx, client, log, currentReq.FleetId)
	if err != nil {
		return EditOutput{}, err
	}

	// create instance
	log.Info("Creating new instance...")
	now := time.Now()
	t := currentReq.ValidUntil.Sub(now).Round(time.Second)
	out, err := helpers.CreateSpotRequest(ctx, client, name, version, opts.MinCPUs, opts.MinMemory, opts.MaxPrice, template, t)
	if err != nil {
		return EditOutput{}, err
	}

	// wait for instance to be running
	newInstance, err := waitInstance(ctx, client, log, out.FleetId)
	if err != nil {
		return EditOutput{}, err
	}

	// wait until port 22 is reachable
	err = helpers.WaitUntilReachable(*newInstance.PublicIpAddress, 22)
	if err != nil {
		return EditOutput{}, err
	}

	// power off devspace
	sshClient, err := ssh.NewSSHClient(*currentInstance.PublicIpAddress, 22, "ec2-user", string(identityKey))
	if err != nil {
		return EditOutput{}, err
	}
	_, err = sshClient.Run("sudo machinectl terminate devspace")
	if err != nil {
		return EditOutput{}, err
	}
	_, err = sshClient.Run("sudo umount /dev/sdf1")
	if err != nil {
		return EditOutput{}, err
	}

	// detach ebs volume
	_, err = helpers.DetachEBSVolume(ctx, client, volumeID)
	if err != nil {
		return EditOutput{}, err
	}

	// wait until ebs volume is detached
	err = helpers.WaitUntilEBSUnattached(ctx, client, volumeID)
	if err != nil {
		return EditOutput{}, err
	}

	// attach ebs volume on new instance
	err = helpers.AttachEBSVolume(ctx, client, *newInstance.InstanceId, volumeID)
	if err != nil {
		return EditOutput{}, err
	}
	log.Info(fmt.Sprintf("Attached EBS volume with id=%s on the new instance", volumeID))

	// terminate old instance
	err = helpers.CancelFleetRequests(ctx, client, []string{*currentReq.FleetId})
	if err != nil {
		log.Error(err)
		return EditOutput{}, err
	}
	log.Info("Terminated old instance with id: ", *currentInstance.InstanceId)
	log.Info("Scaled successfully!")

	return EditOutput{
		InstanceID:     *newInstance.InstanceId,
		InstanceIP:     *newInstance.PublicIpAddress,
		InstanceType:   fmt.Sprint(newInstance.InstanceType),
		FleetRequestID: *out.FleetId,
	}, nil
}
