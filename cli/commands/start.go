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

	loginCommand := fmt.Sprintf("ssh -i <your-key.pem> -p 2222 -o StrictHostKeyChecking=no root@%s", out.PublicIP)

	// create SSH config entry
	configPath, err := util.CreateSSHConfig(*cfg, out.PublicIP, name)
	if err != nil {
		log.Warn(fmt.Sprintf("Error creating SSH config entry for %s: %s", name, err))
	} else {
		log.Info(fmt.Sprintf("Created SSH config entry for %s.", name))
		log.Info(fmt.Sprintf("You can customize the SSH config entry at %s", configPath))
		loginCommand = fmt.Sprintf("ssh -i <your-key.pem> root@%s", name)
	}

	if wait {
		// wait until port 2222 is reachable
		log.Info("Waiting for port 2222 (ssh) to be reachable. This can take a few minutes...")
		err = helpers.WaitUntilReachable(out.PublicIP, out.Port)
		if err != nil {
			return err
		}

		log.Info("You can now ssh into your dev space with the following command: ")
		fmt.Printf("$ %s\n", loginCommand)
	}

	return nil
}
