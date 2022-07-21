package handlers

import (
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/felipemarinho97/invest-path/clients"
)

type Handler struct {
	EC2Client clients.IEC2Client
	SSMClient *ssm.Client
}

func NewHandler(ec2Client clients.IEC2Client, ssmClient *ssm.Client) *Handler {
	return &Handler{
		EC2Client: ec2Client,
		SSMClient: ssmClient,
	}
}
