package cli

import (
	"os"
	"strings"

	"github.com/felipemarinho97/dev-spaces/handlers"
	"github.com/olekukonko/tablewriter"
	"github.com/urfave/cli/v2"
)

func statusCommand(ctx *cli.Context) error {
	h := ctx.Context.Value("handler").(*handlers.Handler)

	items, err := h.Status(ctx.Context, handlers.StatusOptions{
		Name:   ctx.String("name"),
		SortBy: handlers.StatusSortOption(ctx.String("sort-by")),
	})
	if err != nil {
		return err
	}

	data := [][]string{}

	for _, item := range items {
		data = append(data, []string{
			item.Name,
			item.Status.String(),
			item.RequestId,
			item.CreateTime,
			item.ActivityStat,
		})
	}

	table := tablewriter.NewWriter(os.Stdout)
	for _, row := range data {
		if strings.Contains(row[1], "active") {
			table.Rich(row, []tablewriter.Colors{{}, {tablewriter.Normal, tablewriter.FgGreenColor}})
		} else if strings.Contains(row[1], "submitted") || strings.Contains(row[1], "modifying") {
			table.Rich(row, []tablewriter.Colors{{}, {tablewriter.Normal, tablewriter.FgYellowColor}})
		} else {
			table.Rich(row, []tablewriter.Colors{{}, {tablewriter.Normal, tablewriter.FgHiRedColor}})
		}
	}
	table.SetHeader([]string{
		"Name",
		"Request_State",
		"Request_Id",
		"Create_Time",
		"Status",
	})
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
	table.Render()

	return nil
}
