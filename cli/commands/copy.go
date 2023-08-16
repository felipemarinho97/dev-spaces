package commands

import (
	"fmt"

	"github.com/felipemarinho97/dev-spaces/core"
	"github.com/urfave/cli/v2"
)

func CopyCommand(c *cli.Context) error {
	h := c.Context.Value("handler").(*core.Handler)

	name := c.String("name")
	region := c.String("new-region")
	availabilityZone := c.String("availability-zone")

	fmt.Printf("Copying %s to %s\n", name, region)

	out, err := h.Copy(c.Context, core.CopyOptions{
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
