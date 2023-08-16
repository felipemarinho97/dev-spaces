package cli

import (
	"fmt"

	"github.com/felipemarinho97/dev-spaces/handlers"
	"github.com/felipemarinho97/dev-spaces/helpers"
	"github.com/felipemarinho97/dev-spaces/util"
	uuid "github.com/satori/go.uuid"
	"github.com/urfave/cli/v2"
)

func startCommand(c *cli.Context) error {
	ctx := c.Context
	h := ctx.Value("handler").(*handlers.Handler)
	log := h.Logger

	memorySpec := c.Float64("min-memory")
	cpusSpec := c.Int("min-cpus")
	maxPrice := c.String("max-price")
	name := c.String("name")
	if name == "" {
		name = uuid.NewV4().String()
	}
	timeout := c.Duration("timeout")
	minMemory := int(float64(1024) * memorySpec)
	wait := c.Bool("wait")

	ub := util.NewUnknownBar("Starting..")
	ub.Start()
	defer ub.Stop()

	out, err := h.Start(ctx, handlers.StartOptions{
		Name:      name,
		MinCPUs:   cpusSpec,
		MinMemory: minMemory,
		MaxPrice:  maxPrice,
		Timeout:   timeout,
	})
	if err != nil {
		return err
	}

	if wait {
		// wait until port 2222 is reachable
		log.Info("Waiting for port 2222 (ssh) to be reachable. This can take a few minutes...")
		err = helpers.WaitUntilReachable(out.PublicIP, out.Port)
		if err != nil {
			return err
		}
		host := out.CustomDNS
		if host == "" {
			host = out.PublicIP
		}

		log.Info("You can now ssh into your dev space with the following command: ")
		fmt.Printf("$ ssh -i <your-key.pem> -p 2222 -o StrictHostKeyChecking=no root@%s\n", host)
	}

	return nil
}
