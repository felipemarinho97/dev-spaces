package commands

import (
	"github.com/felipemarinho97/dev-spaces/cli/util"
	"github.com/felipemarinho97/dev-spaces/core"
	"github.com/urfave/cli/v2"
)

func DestroyCommand(c *cli.Context) error {
	h := c.Context.Value("handler").(*core.Handler)

	name := c.String("name")

	ub := util.NewUnknownBar("Destroying...")
	ub.Start()
	defer ub.Stop()

	return h.Destroy(c.Context, core.DestroyOptions{
		Name: name,
	})
}
