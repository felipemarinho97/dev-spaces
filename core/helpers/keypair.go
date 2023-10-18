package helpers

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/felipemarinho97/invest-path/clients"
)

func GetKeyPair(ctx context.Context, client clients.IEC2Client, keyName string) (*types.KeyPairInfo, error) {
	keyPairs, err := client.DescribeKeyPairs(ctx, &ec2.DescribeKeyPairsInput{
		KeyNames: []string{keyName},
	})
	if err != nil {
		return nil, err
	}

	if len(keyPairs.KeyPairs) == 0 {
		return nil, fmt.Errorf("no key pair found with name %s", keyName)
	}

	return &keyPairs.KeyPairs[0], nil
}
