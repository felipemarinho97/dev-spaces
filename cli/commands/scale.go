package commands

import (
	"fmt"

	"github.com/felipemarinho97/dev-spaces/cli/clients"
	"github.com/felipemarinho97/dev-spaces/cli/config"
	"github.com/felipemarinho97/dev-spaces/cli/util"
	"github.com/felipemarinho97/dev-spaces/core"
	"github.com/urfave/cli/v2"
)

func EditSpecCommand(c *cli.Context) error {
	h := c.Context.Value("handler").(*core.Handler)
	cfg := c.Context.Value("config").(*config.Config)
	log := h.Logger

	name := c.String("name")
	minCPUs := c.Int("min-cpus")
	minMemory := c.Int("min-memory")
	maxPrice := c.String("max-price")
	identityFile := c.String("identity-file")

	ub := util.NewUnknownBar("Editing..")
	ub.Start()
	defer ub.Stop()

	out, err := h.EditSpec(c.Context, core.EditSpecOptions{
		Name:      name,
		MinCPUs:   minCPUs,
		MinMemory: 1024 * minMemory,
		MaxPrice:  maxPrice,
		SSHKey:    identityFile,
	})
	if err != nil {
		return err
	}

	// add dns record
	customDNS, err := clients.CreateDNSRecord(*cfg, out.InstanceIP, name)
	if err != nil {
		log.Warn(fmt.Printf("Error creating DNS record. This is your new IPv4 address: %s", out.InstanceIP))
		log.Debug(err)
	} else {
		log.Info(fmt.Printf("Updated DNS record: %s -> %s", out.InstanceIP, customDNS))
	}

	return err
}
