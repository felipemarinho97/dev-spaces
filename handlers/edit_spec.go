package handlers

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/felipemarinho97/dev-spaces/util"
	"github.com/felipemarinho97/dev-spaces/util/ssh"
	"github.com/felipemarinho97/invest-path/clients"
	"github.com/samber/lo"
	"gopkg.in/validator.v2"
)

type EditSpecOptions struct {
	// Name of the Dev Space
	Name string `validate:"nonzero"`
	// MinMemory is the amount of memory in MiB
	MinMemory int `validate:"min=0"`
	// MinCPUs is the amount of cpus
	MinCPUs int `validate:"min=0"`
	// MaxPrice is the maximum price for the instance
	MaxPrice string `validate:"nonzero"`
	// SSHKey is the path of the SSH key
	SSHKey string `validate:"nonzero"`
}

type EditOutput struct {
	// InstanceID is the ID of the instance
	InstanceID string `json:"instance_id"`
	// InstanceIP is the Public IP of the instance
	InstanceIP string `json:"instance_ip"`
	// InstanceType is the type of the instance
	InstanceType string `json:"instance_type"`
	// SpotFleetRequestID is the ID of the SpotFleetRequest
	SpotFleetRequestID string `json:"spot_fleet_request_id"`
}

func (h *Handler) EditSpec(ctx context.Context, opts EditSpecOptions) (EditOutput, error) {
	err := validator.Validate(opts)
	if err != nil {
		return EditOutput{}, err
	}

	identityKey, err := util.Readfile(opts.SSHKey)
	if err != nil {
		return EditOutput{}, err
	}

	client := h.EC2Client
	ub := util.NewUnknownBar("Editing...")

	name, version := util.GetTemplateNameAndVersion(opts.Name)
	template, err := getLaunchTemplateByName(ctx, client, name)
	if err != nil {
		return EditOutput{}, err
	}
	volumeID := util.GetTag(template.Tags, "dev-spaces:volume-id")

	// get current spot instance request
	currentReq, err := getCurrentSpotRequest(ctx, client, name)
	if err != nil {
		return EditOutput{}, err
	}
	instanceID, err := waitInstance(client, ctx, currentReq.SpotFleetRequestId, ub)
	if err != nil {
		return EditOutput{}, err
	}
	currentInstance, err := getInstanceData(ctx, client, instanceID)
	if err != nil {
		return EditOutput{}, err
	}

	// create instance
	now := time.Now()
	t := currentReq.SpotFleetRequestConfig.ValidUntil.Sub(now).Round(time.Second)
	out, err := createSpotRequest(ctx, client, name, version, opts.MinCPUs, opts.MinMemory, opts.MaxPrice, template, t)
	if err != nil {
		return EditOutput{}, err
	}

	// wait for instance to be running
	instanceID, err = waitInstance(client, ctx, out.SpotFleetRequestId, ub)
	if err != nil {
		return EditOutput{}, err
	}
	newInstance, err := getInstanceData(ctx, client, instanceID)
	if err != nil {
		return EditOutput{}, err
	}

	// wait until port 22 is reachable
	err = waitUntilReachable(*newInstance.PublicIpAddress, 22)
	if err != nil {
		return EditOutput{}, err
	}

	// power off devspace
	sshClient, err := ssh.NewSSHClient(*currentInstance.PublicIpAddress, 22, "ec2-user", string(identityKey))
	if err != nil {
		return EditOutput{}, err
	}
	o, err := sshClient.Run("sudo machinectl terminate devspace")
	if err != nil {
		return EditOutput{}, err
	}
	fmt.Print(o)
	o, err = sshClient.Run("sudo umount /dev/sdf1")
	if err != nil {
		return EditOutput{}, err
	}
	fmt.Print(o)

	// detach ebs volume
	_, err = detachEBSVolume(ctx, client, volumeID)
	if err != nil {
		return EditOutput{}, err
	}

	// wait until ebs volume is detached
	err = waitUntilEBSUnattached(ctx, h.EC2Client, volumeID)
	if err != nil {
		return EditOutput{}, err
	}

	// attach ebs volume on new instance
	err = attachEBSVolume(ctx, h.EC2Client, *newInstance.InstanceId, volumeID)
	if err != nil {
		return EditOutput{}, err
	}

	// terminate old instance
	_, err = client.CancelSpotFleetRequests(ctx, &ec2.CancelSpotFleetRequestsInput{
		SpotFleetRequestIds: []string{*currentReq.SpotFleetRequestId},
		TerminateInstances:  aws.Bool(true),
	})
	if err != nil {
		fmt.Println(err)
		return EditOutput{}, err
	}

	return EditOutput{
		InstanceID:         *newInstance.InstanceId,
		InstanceIP:         *newInstance.PublicIpAddress,
		InstanceType:       fmt.Sprint(newInstance.InstanceType),
		SpotFleetRequestID: *out.SpotFleetRequestId,
	}, nil
}

func getCurrentSpotRequest(ctx context.Context, client clients.IEC2Client, name string) (*types.SpotFleetRequestConfig, error) {
	requests, err := getSpotRequestStatus(ctx, client, name)
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

func getInstanceData(ctx context.Context, client clients.IEC2Client, instanceID string) (*types.Instance, error) {
	instances, err := client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
		InstanceIds: []string{instanceID},
	})
	if err != nil {
		return nil, err
	}

	if len(instances.Reservations) == 0 {
		return nil, fmt.Errorf("no instance found with ID %s", instanceID)
	}

	if len(instances.Reservations[0].Instances) == 0 {
		return nil, fmt.Errorf("no instance found with ID %s", instanceID)
	}

	return &instances.Reservations[0].Instances[0], nil
}

func waitUntilReachable(ip string, port int) error {
	for {
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", ip, port), 1*time.Second)
		if err == nil {
			conn.Close()
			return nil
		}
	}
}

func detachEBSVolume(ctx context.Context, client clients.IEC2Client, volumeID string) (*ec2.DetachVolumeOutput, error) {
	out, err := client.DetachVolume(ctx, &ec2.DetachVolumeInput{
		VolumeId: aws.String(volumeID),
		Force:    aws.Bool(false),
	})
	if err != nil {
		return nil, err
	}

	return out, nil
}

func waitUntilEBSUnattached(ctx context.Context, client clients.IEC2Client, volumeID string) error {
	for {
		vol, err := client.DescribeVolumes(ctx, &ec2.DescribeVolumesInput{
			VolumeIds: []string{volumeID},
		})
		if err != nil {
			return err
		}
		if len(vol.Volumes) == 0 {
			return fmt.Errorf("no volume found with ID %s", volumeID)
		}
		if len(vol.Volumes[0].Attachments) == 0 || vol.Volumes[0].Attachments == nil {
			return nil
		}
		time.Sleep(1 * time.Second)
	}
}
