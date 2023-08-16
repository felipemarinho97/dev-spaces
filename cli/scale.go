package cli

import (
	"github.com/felipemarinho97/dev-spaces/handlers"
	"github.com/urfave/cli/v2"
)

func editSpecCommand(c *cli.Context) error {
	h := c.Context.Value("handler").(*handlers.Handler)

	name := c.String("name")
	minCPUs := c.Int("min-cpus")
	minMemory := c.Int("min-memory")
	maxPrice := c.String("max-price")
	identityFile := c.String("identity-file")

	_, err := h.EditSpec(c.Context, handlers.EditSpecOptions{
		Name:      name,
		MinCPUs:   minCPUs,
		MinMemory: 1024 * minMemory,
		MaxPrice:  maxPrice,
		SSHKey:    identityFile,
	})

	return err
}
