package commands

import (
	"github.com/felipemarinho97/dev-spaces/cli/util"
	"github.com/felipemarinho97/dev-spaces/core"
	"github.com/urfave/cli/v2"
)

func StopCommand(ctx *cli.Context) error {
	h := ctx.Context.Value("handler").(*core.Handler)
	log := h.Logger
	name := ctx.String("name")

	ub := util.NewUnknownBar("Stopping...")
	ub.Start()
	defer ub.Stop()

	_, err := h.Stop(ctx.Context, core.StopOptions{
		Name: name,
	})
	if err != nil {
		return err
	}

	log.Info("Stopped")
	return nil
}
