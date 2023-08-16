package handlers

import (
	"context"

	"github.com/felipemarinho97/dev-spaces/helpers"
)

type StopOptions struct {
	// Name of the dev space
	Name string
}

type StopOutput struct {
	// Quantity of instances that were stopped
	Quantity int
}

func (h *Handler) Stop(ctx context.Context, opts StopOptions) (StopOutput, error) {
	name := opts.Name

	client := h.EC2Client
	log := h.Logger

	qnt, err := helpers.CancelSpotRequests(ctx, client, log, name)
	if err != nil {
		return StopOutput{}, err
	}

	return StopOutput{Quantity: qnt}, nil
}
