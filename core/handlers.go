package core

import (
	"github.com/felipemarinho97/dev-spaces/core/log"
	"github.com/felipemarinho97/invest-path/clients"
)

type Config struct {
	DefaultRegion string
}

type Handler struct {
	EC2Client clients.IEC2Client
	Logger    log.Logger
	Config    Config
}

func NewHandler(cfg Config, ec2Client clients.IEC2Client, logger log.Logger) *Handler {
	return &Handler{
		EC2Client: ec2Client,
		Logger:    logger,
		Config:    cfg,
	}
}
