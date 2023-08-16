package helpers

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/felipemarinho97/invest-path/clients"
)

func CreateSnapshot(ctx context.Context, client clients.IEC2Client, volumeID string) (string, error) {
	snapshot, err := client.CreateSnapshot(ctx, &ec2.CreateSnapshotInput{
		VolumeId: aws.String(volumeID),
	})
	if err != nil {
		return "", err
	}

	return *snapshot.SnapshotId, nil
}

func WaitForSnapshot(ctx context.Context, client clients.IEC2Client, snapshotID string) error {
	for {
		snapshot, err := client.DescribeSnapshots(ctx, &ec2.DescribeSnapshotsInput{
			SnapshotIds: []string{snapshotID},
		})
		if err != nil {
			return err
		}

		if snapshot.Snapshots[0].State == types.SnapshotStateCompleted {
			return nil
		}

		if snapshot.Snapshots[0].State == types.SnapshotStateError {
			return fmt.Errorf("snapshot %s is in error state: %s", snapshotID, *snapshot.Snapshots[0].StateMessage)
		}

		time.Sleep(time.Second * 1)
	}
}
