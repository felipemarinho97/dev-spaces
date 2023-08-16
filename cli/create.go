package cli

import (
	"fmt"

	"github.com/felipemarinho97/dev-spaces/handlers"
	"github.com/felipemarinho97/dev-spaces/util"
	"github.com/urfave/cli/v2"
)

func createCommand(c *cli.Context) error {
	h := c.Context.Value("handler").(*handlers.Handler)
	log := h.Logger

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

	ub := util.NewUnknownBar("Creating...")
	ub.Start()
	defer ub.Stop()

	_, err = h.Create(c.Context, handlers.CreateOptions{
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
	if err != nil {
		return err
	}

	log.Info(fmt.Sprintf("DevSpace \"%s\" created successfully.", name))
	return nil
}
