package util

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/go-playground/validator/v10"
	"gopkg.in/yaml.v2"
)

var (
	Validator = validator.New()
)

func IsManaged(tags []types.Tag) bool {
	for _, tag := range tags {
		if *tag.Key == "managed-by" && *tag.Value == "dev-spaces" {
			return true
		}
	}

	return false
}

func IsDevSpace(tags []types.Tag, devSpaceName string) bool {

	for _, tag := range tags {
		if *tag.Key == "dev-spaces:name" && *tag.Value == devSpaceName {
			return true
		}
	}

	// empty dev space name matches any dev space
	if IsManaged(tags) && devSpaceName == "" {
		return true
	}

	return false
}

func GetTag(tags []types.Tag, key string) string {
	for _, tag := range tags {
		if *tag.Key == key {
			return *tag.Value
		}
	}

	return ""
}

func loadFile(filename string) ([]byte, error) {
	// check if location is a url
	if strings.HasPrefix(filename, "http") {
		return downloadFrom(filename)
	}

	return ioutil.ReadFile(filename)
}

func downloadFrom(filename string) ([]byte, error) {
	// Get the data
	resp, err := http.Get(filename)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	// Read the data
	return ioutil.ReadAll(resp.Body)
}

func Readfile(filename string) (string, error) {
	b, err := loadFile(filename)
	if err != nil {
		return "", err
	}

	return string(b), nil
}

func GetValue(ptr *string) string {
	if ptr == nil {
		return ""
	}

	return *ptr
}

func LoadYAML(filename string, config interface{}) (err error) {
	file, err := loadFile(filename)
	if err != nil {
		return
	}

	err = yaml.Unmarshal(file, config)
	if err != nil {
		return
	}

	return
}

func GetTemplateNameAndVersion(name string) (string, string) {
	if name == "" {
		return "", ""
	}

	parts := strings.Split(name, "/")
	if len(parts) == 1 {
		return parts[0], "$Default"
	}

	return parts[0], parts[1]
}

func GenerateTags(templateName string) []types.Tag {
	return []types.Tag{
		{
			Key:   aws.String("managed-by"),
			Value: aws.String("dev-spaces"),
		},
		{
			Key:   aws.String("dev-spaces:name"),
			Value: aws.String(templateName),
		},
		{
			Key:   aws.String("Name"),
			Value: aws.String(templateName),
		},
	}
}

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
