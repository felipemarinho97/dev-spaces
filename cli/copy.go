package cli

import (
	"fmt"

	"github.com/felipemarinho97/dev-spaces/handlers"
	"github.com/urfave/cli/v2"
)

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
