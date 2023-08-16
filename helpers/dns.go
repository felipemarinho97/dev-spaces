package helpers

import (
	"fmt"
	"net/http"

	"github.com/felipemarinho97/dev-spaces/config"
)

func CreateDNSRecord(config config.Config, ip, name string) (string, error) {
	endpoint := config.DNS.Endpoint
	token := config.DNS.Token

	// perform http request to provider
	client := &http.Client{}
	req, err := http.NewRequest("GET", fmt.Sprintf("%s?ip=%s&domains=%s&token=%s", endpoint, ip, name, token), nil)
	if err != nil {
		return "", err
	}

	// add token to request
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))

	// perform request
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}

	// check response status code
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("failed to update dns: %s", resp.Status)
	}

	return fmt.Sprintf("%s.%s", name, config.DNS.Domain), nil
}
