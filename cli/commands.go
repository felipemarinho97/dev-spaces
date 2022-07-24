package cli

import (
	"fmt"

	"github.com/felipemarinho97/dev-spaces/handlers"
	"github.com/felipemarinho97/dev-spaces/util"
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
	devSpaceAMIID, err := util.ParseAMIFilter(c.String("ami"))
	if err != nil {
		return err
	}
	customHostAMIID, err := util.ParseAMIFilter(c.String("custom-host-ami"))
	if err != nil {
		return err
	}
	startupScript := c.String("custom-startup-script")
	preferedInstanceType := c.String("prefered-instance-type")
	securityGroupIds := c.StringSlice("security-group-ids")
	storageSize := c.Int("storage-size")
	spec, err := util.ParseInstanceSpec(preferedInstanceType)
	if err != nil {
		return err
	}

	var hostAMI *handlers.AMIFilter
	if (customHostAMIID != util.AMIFilter{}) {
		hostAMI = &handlers.AMIFilter{
			ID:    customHostAMIID.ID,
			Name:  customHostAMIID.Name,
			Arch:  customHostAMIID.Arch,
			Owner: customHostAMIID.Owner,
		}
	}

	return h.Create(c.Context, handlers.CreateOptions{
		Name:               name,
		KeyName:            keyName,
		InstanceProfileArn: instanceProfileArn,
		DevSpaceAMI: handlers.AMIFilter{
			ID:    devSpaceAMIID.ID,
			Name:  devSpaceAMIID.Name,
			Arch:  devSpaceAMIID.Arch,
			Owner: devSpaceAMIID.Owner,
		},
		HostAMI:           hostAMI,
		StartupScriptPath: startupScript,
		PreferedLaunchSpecs: handlers.PreferedLaunchSpecs{
			InstanceType: handlers.InstanceType(spec.InstanceType),
			MinMemory:    spec.MinMemory,
			MinCPU:       spec.MinCPU,
		},
		SecurityGroupIds: securityGroupIds,
		StorageSize:      storageSize,
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

func copyCommand(c *cli.Context) error {
	h := c.Context.Value("handler").(*handlers.Handler)

	name := c.String("name")
	region := c.String("new-region")
	availabilityZone := c.String("availability-zone")

	fmt.Printf("Copying %s to %s\n", name, region)

	out, err := h.Copy(c.Context, handlers.CopyOptions{
		Name:             name,
		Region:           region,
		AvailabilityZone: availabilityZone,
	})
	if err != nil {
		return err
	}

	fmt.Printf("launch-template-id=%s\n", out.LaunchTemplateID)
	fmt.Printf("volume-id=%s\n", out.VolumeID)

	return nil
}

func destroyCommand(c *cli.Context) error {
	h := c.Context.Value("handler").(*handlers.Handler)

	name := c.String("name")

	return h.Destroy(c.Context, handlers.DestroyOptions{
		Name: name,
	})
}
