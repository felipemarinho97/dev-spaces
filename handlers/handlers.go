package handlers

import (
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/felipemarinho97/dev-spaces/log"
	"github.com/felipemarinho97/invest-path/clients"
)

type Handler struct {
	Region    string
	EC2Client clients.IEC2Client
	Logger    log.Logger
}

func NewHandler(region string, ec2Client clients.IEC2Client, ssmClient *ssm.Client, logger log.Logger) *Handler {
	return &Handler{
		Region:    region,
		EC2Client: ec2Client,
		Logger:    logger,
	}
}
