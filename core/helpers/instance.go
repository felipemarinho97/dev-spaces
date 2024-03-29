package helpers

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/felipemarinho97/dev-spaces/core/util"
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
		time.Sleep(1 * time.Second)
	}
}

func WaitForFleetInstance(ctx context.Context, client clients.IEC2Client, requestID string, wantedState types.InstanceStateName) (string, error) {
	for {
		time.Sleep(time.Second * 1)
		out2, err := client.DescribeFleetInstances(ctx, &ec2.DescribeFleetInstancesInput{
			FleetId: &requestID,
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

		// get fleet history
		out3, err := client.DescribeFleetHistory(ctx, &ec2.DescribeFleetHistoryInput{
			FleetId:   &requestID,
			StartTime: aws.Time(time.Now().Add(-24 * time.Hour)),
		})
		if err != nil {
			continue
		}

		if len(out3.HistoryRecords) > 0 {
			// iterate over history records, if there is a record with a status of error, return error and print error message
			for _, record := range out3.HistoryRecords {
				if record.EventType == "error" && record.EventInformation.EventDescription != nil {
					return "", fmt.Errorf("error creating instance: %s", *record.EventInformation.EventDescription)
				}
			}
		}
	}
}
