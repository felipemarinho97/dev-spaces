package helpers

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/felipemarinho97/dev-spaces/util"
	"github.com/felipemarinho97/invest-path/clients"
)

func GetInstanceData(ctx context.Context, client clients.IEC2Client, instanceID string) (*types.Instance, error) {
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

func GetManagedInstances(ctx context.Context, client clients.IEC2Client) (map[string]*types.Instance, error) {
	instances, err := client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("tag:managed-by"),
				Values: []string{"dev-spaces"},
			},
		},
	})
	if err != nil {
		return nil, err
	}

	// map instance by space name
	var managedInstances = make(map[string]*types.Instance)
	for _, instance := range instances.Reservations {
		for _, i := range instance.Instances {
			name := util.GetTag(i.Tags, "dev-spaces:name")

			if inst := managedInstances[name]; inst != nil {
				if inst.LaunchTime.After(*i.LaunchTime) {
					continue
				}
			}
			managedInstances[name] = &i
		}
	}

	return managedInstances, nil
}

func WaitUntilReachable(host string, port int) error {
	for {
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", host, port), 1*time.Second)
		if err == nil {
			conn.Close()
			return nil
		}
	}
}

func WaitForSpotFleetInstance(ctx context.Context, client clients.IEC2Client, requestID string, wantedState types.InstanceStateName) (string, error) {
	for {
		time.Sleep(time.Second * 1)
		out2, err := client.DescribeSpotFleetInstances(ctx, &ec2.DescribeSpotFleetInstancesInput{
			SpotFleetRequestId: &requestID,
		})
		if err != nil {
			fmt.Println(err)
			continue
		}

		if len(out2.ActiveInstances) == 0 && wantedState == types.InstanceStateNameTerminated {
			return "", nil
		}

		for _, s := range out2.ActiveInstances {
			time.Sleep(time.Second * 1)
			instanceData, err := GetInstanceData(ctx, client, *s.InstanceId)
			if err != nil {
				// print unicode X to indicate error
				fmt.Printf("\x1b[31m%s\x1b[0m\n", "\u2717")
				continue
			}

			if instanceData.State.Name == wantedState {
				return *instanceData.InstanceId, nil
			}

		}
	}
}
