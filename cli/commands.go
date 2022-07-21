package cli

import (
	"github.com/felipemarinho97/dev-spaces/handlers"
	uuid "github.com/satori/go.uuid"
	"github.com/urfave/cli/v2"
)

func startCommand(c *cli.Context) error {
	ctx := c.Context
	h := ctx.Value("handler").(*handlers.Handler)
	memorySpec := c.Float64("min-memory")
	cpusSpec := c.Int("min-cpus")
	maxPrice := c.String("max-price")
	name := c.String("name")
	if name == "" {
		name = uuid.NewV4().String()
	}
	timeout := c.Duration("timeout")
	minMemory := int(float64(1024) * memorySpec)
	return h.Start(ctx, handlers.StartOptions{
		Name:      name,
		MinCPUs:   cpusSpec,
		MinMemory: minMemory,
		MaxPrice:  maxPrice,
		Timeout:   timeout,
	})
}

func stopCommand(ctx *cli.Context) error {
	h := ctx.Context.Value("handler").(*handlers.Handler)
	name := ctx.String("name")

	return h.Stop(ctx.Context, handlers.StopOptions{
		Name: name,
	})
}

func statusCommand(ctx *cli.Context) error {
	h := ctx.Context.Value("handler").(*handlers.Handler)

	return h.Status(ctx.Context, handlers.StartOptions{
		Name: ctx.String("name"),
	})
}

func createCommand(c *cli.Context) error {
	h := c.Context.Value("handler").(*handlers.Handler)

	name := c.String("name")
	keyName := c.String("key-name")
	instanceProfileArn := c.String("instance-profile-arn")
	devSpaceAMIID := c.String("ami")
	customHostAMIID := c.String("custom-host-ami")
	startupScript := c.String("custom-startup-script")
	preferedInstanceType := c.String("prefered-instance-type")
	securityGroupIds := c.StringSlice("security-group-ids")
	storageSize := c.Int("storage-size")

	return h.Create(c.Context, handlers.CreateOptions{
		Name:                 name,
		KeyName:              keyName,
		InstanceProfileArn:   instanceProfileArn,
		DevSpaceAMIID:        devSpaceAMIID,
		HostAMIID:            customHostAMIID,
		StartupScript:        startupScript,
		PreferedInstanceType: handlers.InstanceType(preferedInstanceType),
		SecurityGroupIds:     securityGroupIds,
		StorageSize:          storageSize,
	})
}

func listCommand(ctx *cli.Context) error {
	h := ctx.Context.Value("handler").(*handlers.Handler)

	return h.ListTemplates(ctx.Context, handlers.ListOptions{
		Output: handlers.OutputFormat(ctx.String("output")),
	})
}

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
