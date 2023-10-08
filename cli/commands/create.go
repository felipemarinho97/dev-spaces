package commands

import (
	"fmt"
	"os"
	"os/signal"
	"strings"

	"github.com/felipemarinho97/dev-spaces/cli/util"
	"github.com/felipemarinho97/dev-spaces/core"
	"github.com/urfave/cli/v2"
)

func CreateCommand(c *cli.Context) error {
	// if there is a signal CTRL+C, destroy the devspace
	handleSignal(c)

	h := c.Context.Value("handler").(*core.Handler)
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

	var hostAMI *core.AMIFilter
	if (customHostAMIID != util.AMIFilter{}) {
		hostAMI = &core.AMIFilter{
			ID:    customHostAMIID.ID,
			Name:  customHostAMIID.Name,
			Arch:  customHostAMIID.Arch,
			Owner: customHostAMIID.Owner,
		}
	}

	ub := util.NewUnknownBar("Creating...")
	ub.Start()
	defer ub.Stop()

	_, err = h.Create(c.Context, core.CreateOptions{
		Name:               name,
		KeyName:            keyName,
		InstanceProfileArn: instanceProfileArn,
		DevSpaceAMI: core.AMIFilter{
			ID:    devSpaceAMIID.ID,
			Name:  devSpaceAMIID.Name,
			Arch:  devSpaceAMIID.Arch,
			Owner: devSpaceAMIID.Owner,
		},
		HostAMI:           hostAMI,
		StartupScriptPath: startupScript,
		PreferedLaunchSpecs: core.PreferedLaunchSpecs{
			InstanceType: core.InstanceType(spec.InstanceType),
			MinMemory:    spec.MinMemory,
			MinCPU:       spec.MinCPU,
		},
		SecurityGroupIds: securityGroupIds,
		StorageSize:      storageSize,
	})
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			log.Warn(fmt.Sprintf("DevSpace \"%s\" already exists.", name))
			return err
		}
		return err
	}

	log.Info(fmt.Sprintf("DevSpace \"%s\" created successfully.", name))
	return nil
}

func handleSignal(c *cli.Context) {
	s := make(chan os.Signal, 1)
	signal.Notify(s, os.Interrupt)
	go func() {
		for range s {
			DestroyCommand(c)
			os.Exit(0)
		}
	}()
}
