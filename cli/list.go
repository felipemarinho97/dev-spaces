package cli

import (
	"fmt"
	"os"

	"github.com/felipemarinho97/dev-spaces/handlers"
	"github.com/olekukonko/tablewriter"
	"github.com/urfave/cli/v2"
)

func listCommand(ctx *cli.Context) error {
	h := ctx.Context.Value("handler").(*handlers.Handler)
	output := handlers.OutputFormat(ctx.String("output"))

	items, err := h.ListSpaces(ctx.Context, handlers.ListOptions{})
	if err != nil {
		return err
	}

	table := tablewriter.NewWriter(os.Stdout)
	header := []string{"Space Name", "Ver", "ID", "Create Time"}
	if output == "wide" {
		extra_headers := []string{"Instance ID", "Instance Type", "Instance State", "Public DNS", "Public IP", "Key Name", "Zone"}
		header = append(header, extra_headers...)
	}
	table.SetHeader(header)
	table.SetAutoWrapText(false)
	table.SetAutoFormatHeaders(true)
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetCenterSeparator("")
	table.SetColumnSeparator("")
	table.SetRowSeparator("")
	table.SetHeaderLine(true)
	table.SetBorder(false)
	table.SetTablePadding("\t") // pad with tabs
	table.SetNoWhiteSpace(true)

	for _, item := range items {
		row := []string{}
		row = append(row,
			item.Name,
			fmt.Sprint(item.Version),
			item.LaunchTemplateID,
			item.CreateTime,
		)

		if output == "wide" {
			row = append(row,
				item.InstanceID,
				item.InstanceType,
				item.InstanceState,
				item.PublicDNS,
				item.PublicIP,
				item.KeyName,
				item.Zone,
			)
		}

		table.Append(row)
	}

	table.Render()

	return nil
}
