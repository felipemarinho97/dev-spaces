package handlers

import (
	"context"
	"sort"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/felipemarinho97/dev-spaces/helpers"
	"github.com/felipemarinho97/dev-spaces/util"
)

type Status types.BatchState
type StatusSortOption string

const (
	StatusSortByName   StatusSortOption = "name"
	StatusSortByStatus StatusSortOption = "status"
	StatusSortByTime   StatusSortOption = "time"
)

type StatusOptions struct {
	// Name of the dev space
	Name string
	// SortBy is the column to sort by
	SortBy StatusSortOption
}

type StatusItem struct {
	Name         string
	Status       Status
	RequestId    string
	CreateTime   string
	ActivityStat string
}

func (h *Handler) Status(ctx context.Context, opts StatusOptions) ([]StatusItem, error) {
	name := opts.Name
	client := h.EC2Client

	requests, err := helpers.GetFleetStatus(ctx, client, name)
	if err != nil {
		return nil, err
	}

	items := toStatusItem(requests)

	// apply sort
	sort.Slice(items, func(i, j int) bool {
		switch opts.SortBy {
		case StatusSortByName:
			return items[i].Name < items[j].Name
		case StatusSortByStatus:
			return items[i].Status < items[j].Status
		case StatusSortByTime:
			return items[i].CreateTime < items[j].CreateTime
		default:
			return items[i].CreateTime < items[j].CreateTime
		}
	})

	return items, nil
}

func toStatusItem(fleetData []types.FleetData) []StatusItem {
	data := []StatusItem{}
	for _, fleet := range fleetData {
		data = append(data, StatusItem{
			Name:         util.GetTag(fleet.Tags, "dev-spaces:name"),
			Status:       Status(fleet.FleetState),
			RequestId:    getOrNone(fleet.FleetId),
			CreateTime:   string(fleet.CreateTime.Local().Format(time.RFC3339)),
			ActivityStat: string(fleet.ActivityStatus),
		})
	}

	return data
}

func (s *Status) String() string {
	return string(*s)
}
