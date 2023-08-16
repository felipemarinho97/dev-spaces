package util

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type InstanceSpec struct {
	MinMemory    int32
	MinCPU       int32
	InstanceType string
}

func ParseInstanceSpec(spec string) (InstanceSpec, error) {
	// spec format: "mem:0.5,cpus:1,type:t2.micro"
	if spec == "" {
		return InstanceSpec{}, nil
	}
	// check if it's a valid spec with regex
	re := regexp.MustCompile(`^(mem:(\d+\.\d+)|cpus:(\d+)|type:([\w\.]+)|,|[\w\.]+)+$`)
	if !re.MatchString(spec) {
		fmt.Println("Invalid instance spec:", spec)
		return InstanceSpec{}, fmt.Errorf("invalid instance spec: %s", spec)
	}
	parts := strings.Split(spec, ",")
	var instanceSpec InstanceSpec

	for _, part := range parts {
		keyValue := strings.Split(part, ":")

		switch keyValue[0] {
		case "mem":
			mem, _ := strconv.ParseFloat(keyValue[1], 64)
			instanceSpec.MinMemory = int32(float32(mem) * 1024)
		case "cpus":
			cpus, _ := strconv.Atoi(keyValue[1])
			instanceSpec.MinCPU = int32(cpus)
		case "type":
			instanceSpec.InstanceType = keyValue[1]
		default:
			return InstanceSpec{
				InstanceType: keyValue[0],
			}, nil
		}
	}

	return instanceSpec, nil
}

type AMIFilter struct {
	ID    string
	Arch  string
	Name  string
	Owner string
}

func ParseAMIFilter(filter string) (AMIFilter, error) {
	if filter == "" {
		return AMIFilter{}, nil
	}
	// check if match ami id format -> ami-04681a1dbd79675a5
	re := regexp.MustCompile(`^ami-([a-z0-9]+)$`)
	if re.MatchString(filter) {
		return AMIFilter{ID: filter}, nil
	}
	// filter format: "id:ami-12345678,arch:x86_64,name:my-ami,owner:123456789012"
	// check if it's a valid filter with regex
	re = regexp.MustCompile(`^(id:([a-z0-9]{8})|arch:([\w\.]+)|name:([\w\.\-\*]+)|owner:([\w\.]+)|,)+$`)
	if !re.MatchString(filter) {
		fmt.Println("Invalid AMI filter:", filter)
		return AMIFilter{}, fmt.Errorf("invalid AMI filter: %s", filter)
	}
	parts := strings.Split(filter, ",")
	var amiFilter AMIFilter

	for _, part := range parts {
		keyValue := strings.Split(part, ":")

		switch keyValue[0] {
		case "id":
			amiFilter.ID = keyValue[1]
		case "arch":
			amiFilter.Arch = keyValue[1]
		case "name":
			amiFilter.Name = keyValue[1]
		case "owner":
			amiFilter.Owner = keyValue[1]
		default:
			return AMIFilter{}, fmt.Errorf("invalid AMI filter: %s", filter)
		}
	}

	return amiFilter, nil
}
