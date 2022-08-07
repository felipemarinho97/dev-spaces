package handlers

import (
	"context"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/felipemarinho97/dev-spaces/helpers"
	"github.com/felipemarinho97/dev-spaces/util"
	"github.com/olekukonko/tablewriter"
)

type StatusOptions struct {
	// Name of the dev space
	Name string
}

func (h *Handler) Status(ctx context.Context, opts StartOptions) error {
	name := opts.Name
	client := h.EC2Client

	requests, err := helpers.GetFleetStatus(ctx, client, name)
	if err != nil {
		return err
	}

	data := [][]string{}

	for _, request := range requests {
		data = append(data, []string{
			util.GetTag(request.Tags, "dev-spaces:name"),
			string(request.FleetState),
			*request.FleetId,
			string(request.CreateTime.Local().Format(time.RFC3339)),
			string(request.ActivityStatus),
		})
	}
	sort.Slice(data, func(i, j int) bool {
		di := data[i][3]
		dj := data[j][3]
		return di > dj
	})

	table := tablewriter.NewWriter(os.Stdout)
	for _, row := range data {
		if strings.Contains(row[1], string(types.BatchStateActive)) {
			table.Rich(row, []tablewriter.Colors{{}, {tablewriter.Normal, tablewriter.FgGreenColor}})
		} else if strings.Contains(row[1], string(types.BatchStateSubmitted)) || strings.Contains(row[1], string(types.BatchStateModifying)) {
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
