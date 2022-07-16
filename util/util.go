package util

import (
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go/aws"
	"gopkg.in/yaml.v2"
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
	if devSpaceName == "" {
		return true
	}

	for _, tag := range tags {
		if *tag.Key == "dev-spaces:name" && *tag.Value == devSpaceName {
			return true
		}
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
		return parts[0], "1"
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
	}
}
