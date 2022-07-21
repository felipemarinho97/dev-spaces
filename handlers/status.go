package handlers

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/felipemarinho97/dev-spaces/util"
	"github.com/felipemarinho97/invest-path/clients"
	"github.com/olekukonko/tablewriter"
)

type StatusOptions struct {
	// Name of the dev space
	Name string
}

func (h *Handler) Status(ctx context.Context, opts StartOptions) error {
	name := opts.Name
	client := h.EC2Client

	requests, err := getSpotRequestStatus(ctx, client, name)
	if err != nil {
		return err
	}

	data := [][]string{}

	for _, request := range requests {
		data = append(data, []string{
			util.GetTag(request.Tags, "dev-spaces:name"),
			string(request.SpotFleetRequestState),
			*request.SpotFleetRequestId,
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

func getSpotRequestStatus(ctx context.Context, client clients.IEC2Client, name string) ([]types.SpotFleetRequestConfig, error) {
	requests, err := client.DescribeSpotFleetRequests(ctx, &ec2.DescribeSpotFleetRequestsInput{})
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	var filteredRequests []types.SpotFleetRequestConfig
	for _, request := range requests.SpotFleetRequestConfigs {
		if util.IsManaged(request.Tags) && util.IsDevSpace(request.Tags, name) {
			filteredRequests = append(filteredRequests, request)
		}
	}

	return filteredRequests, nil
}
