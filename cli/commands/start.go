package commands

import (
	"fmt"

	"github.com/felipemarinho97/dev-spaces/cli/config"
	"github.com/felipemarinho97/dev-spaces/cli/util"
	"github.com/felipemarinho97/dev-spaces/core"
	"github.com/felipemarinho97/dev-spaces/core/helpers"
	uuid "github.com/satori/go.uuid"
	"github.com/urfave/cli/v2"
)

func StartCommand(c *cli.Context) error {
	ctx := c.Context
	h := ctx.Value("handler").(*core.Handler)
	cfg := ctx.Value("config").(*config.Config)
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

	out, err := h.Start(ctx, core.StartOptions{
		Name:      name,
		MinCPUs:   cpusSpec,
		MinMemory: minMemory,
		MaxPrice:  maxPrice,
		Timeout:   timeout,
	})
	if err != nil {
		return err
	}

	// add dns record
	customDNS, err := util.CreateDNSRecord(*cfg, out.PublicIP, name)
	if err != nil {
		log.Warn(fmt.Printf("Error creating DNS record: %s. Falling back to IPv4 address: %s", err, out.PublicIP))
	} else {
		log.Info(fmt.Printf("Created DNS record: %s -> %s", out.PublicIP, customDNS))
	}

	if wait {
		// wait until port 2222 is reachable
		log.Info("Waiting for port 2222 (ssh) to be reachable. This can take a few minutes...")
		err = helpers.WaitUntilReachable(out.PublicIP, out.Port)
		if err != nil {
			return err
		}
		host := customDNS
		if host == "" {
			host = out.PublicIP
		}

		log.Info("You can now ssh into your dev space with the following command: ")
		fmt.Printf("$ ssh -i <your-key.pem> -p 2222 -o StrictHostKeyChecking=no root@%s\n", host)
	}

	return nil
}
