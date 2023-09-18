package commands

import (
	"github.com/felipemarinho97/dev-spaces/cli/config"
	"github.com/felipemarinho97/dev-spaces/cli/util"
	"github.com/felipemarinho97/dev-spaces/core"
	"github.com/urfave/cli/v2"
)

func EditSpecCommand(c *cli.Context) error {
	h := c.Context.Value("handler").(*core.Handler)
	cfg := c.Context.Value("config").(*config.Config)

	name := c.String("name")
	minCPUs := c.Int("min-cpus")
	minMemory := c.Int("min-memory")
	maxPrice := c.String("max-price")
	identityFile := c.String("identity-file")

	ub := util.NewUnknownBar("Editing..")
	ub.Start()
	defer ub.Stop()

	newSpec, err := h.EditSpec(c.Context, core.EditSpecOptions{
		Name:      name,
		MinCPUs:   minCPUs,
		MinMemory: 1024 * minMemory,
		MaxPrice:  maxPrice,
		SSHKey:    identityFile,
	})
	if err != nil {
		return err
	}

	// update SSH config entry
	_, err = util.CreateSSHConfig(*cfg, newSpec.InstanceIP, name)
	if err != nil {
		h.Logger.Warn("Error updating SSH config entry: %s", err)
	} else {
		h.Logger.Info("Updated SSH config entry")
	}

	return err
}
