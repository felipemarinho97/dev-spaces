package util

import (
	"io"
	"net/http"
	"os"
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

	return os.ReadFile(filename)
}

func downloadFrom(filename string) ([]byte, error) {
	// Get the data
	resp, err := http.Get(filename)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	// Read the data
	return io.ReadAll(resp.Body)
}

func RetrieveFile(filename string) (string, error) {
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

func Map[A any, B any](input []A, m func(A) B) []B {
	output := make([]B, len(input))
	for i, element := range input {
		output[i] = m(element)
	}
	return output
}
