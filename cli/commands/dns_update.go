package commands

import (
	"fmt"

	"github.com/felipemarinho97/dev-spaces/cli/config"
	"github.com/felipemarinho97/dev-spaces/cli/services"
	"github.com/felipemarinho97/dev-spaces/cli/util"
	"github.com/felipemarinho97/dev-spaces/core"
	"github.com/urfave/cli/v2"
)

func DNSUpdate(c *cli.Context) error {
	ctx := c.Context
	h := c.Context.Value("handler").(*core.Handler)
	cnf := c.Context.Value("config").(*config.Config)

	name := c.String("name")

	ub := util.NewUnknownBar("Updating...")
	ub.Start()
	defer ub.Stop()

	// get the dev space
	devSpaces, err := h.ListSpaces(ctx, core.ListOptions{
		Name: name,
	})
	if err != nil {
		return err
	}

	if len(devSpaces) == 0 {
		return fmt.Errorf("dev space %s not found", name)
	}

	devSpace := devSpaces[0]

	// update the DNS
	dns, err := services.CreateDNSRecord(*cnf, devSpace.PublicIP, name)
	if err != nil {
		return err
	}

	fmt.Printf("DNS updated: %s\n", dns)

	return nil
}
