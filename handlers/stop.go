package handlers

import (
	"context"

	"github.com/felipemarinho97/dev-spaces/helpers"
	"github.com/felipemarinho97/dev-spaces/util"
)

type StopOptions struct {
	// Name of the dev space
	Name string
}

func (h *Handler) Stop(ctx context.Context, opts StopOptions) error {
	name := opts.Name
	ub := util.NewUnknownBar("Stopping...")
	ub.Start()
	defer ub.Stop()

	client := h.EC2Client
	log := h.Logger

	err := helpers.CancelSpotRequest(ctx, client, log, name)
	if err != nil {
		return err
	}

	err = helpers.CleanElasticIPs(ctx, client, name)
	if err != nil {
		return err
	}
	log.Info("Stopped")
	return nil
}
