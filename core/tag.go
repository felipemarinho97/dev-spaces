package core

import (
	"context"
)

type TagOptions struct {
	// Name of the dev space
}

type TagOutput struct {
}

func (h *Handler) CreateTag(ctx context.Context, opts TagOptions) (TagOutput, error) {
	return TagOutput{}, nil
}
