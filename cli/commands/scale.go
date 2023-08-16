package commands

import (
	"github.com/felipemarinho97/dev-spaces/cli/util"
	"github.com/felipemarinho97/dev-spaces/core"
	"github.com/urfave/cli/v2"
)

func EditSpecCommand(c *cli.Context) error {
	h := c.Context.Value("handler").(*core.Handler)

	name := c.String("name")
	minCPUs := c.Int("min-cpus")
	minMemory := c.Int("min-memory")
	maxPrice := c.String("max-price")
	identityFile := c.String("identity-file")

	ub := util.NewUnknownBar("Editing..")
	ub.Start()
	defer ub.Stop()

	_, err := h.EditSpec(c.Context, core.EditSpecOptions{
		Name:      name,
		MinCPUs:   minCPUs,
		MinMemory: 1024 * minMemory,
		MaxPrice:  maxPrice,
		SSHKey:    identityFile,
	})

	return err
}
