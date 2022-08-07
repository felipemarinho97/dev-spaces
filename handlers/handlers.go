package handlers

import (
	"github.com/felipemarinho97/dev-spaces/config"
	"github.com/felipemarinho97/dev-spaces/log"
	"github.com/felipemarinho97/invest-path/clients"
)

type Handler struct {
	EC2Client clients.IEC2Client
	Logger    log.Logger
	Config    *config.Config
}

func NewHandler(cfg *config.Config, ec2Client clients.IEC2Client, logger log.Logger) *Handler {
	return &Handler{
		EC2Client: ec2Client,
		Logger:    logger,
		Config:    cfg,
	}
}
