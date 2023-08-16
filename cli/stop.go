package cli

import (
	"github.com/felipemarinho97/dev-spaces/handlers"
	"github.com/felipemarinho97/dev-spaces/util"
	"github.com/urfave/cli/v2"
)

func stopCommand(ctx *cli.Context) error {
	h := ctx.Context.Value("handler").(*handlers.Handler)
	log := h.Logger
	name := ctx.String("name")

	ub := util.NewUnknownBar("Stopping...")
	ub.Start()
	defer ub.Stop()

	_, err := h.Stop(ctx.Context, handlers.StopOptions{
		Name: name,
	})
	if err != nil {
		return err
	}

	log.Info("Stopped")
	return nil
}
