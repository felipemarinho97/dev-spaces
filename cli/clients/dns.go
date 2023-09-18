package clients

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/felipemarinho97/dev-spaces/cli/config"
)

func CreateDNSRecord(config config.Config, ip, name string) (string, error) {
	endpoint := config.DNS.Endpoint
	token := config.DNS.Token

	// if is devspaces.online, get the access token
	if strings.Contains(endpoint, "devspaces.online") {
		var err error
		token, err = GetAccessToken(config)
		if err != nil {
			return "", err
		}
	}

	domain := name

	// perform http request to provider
	client := &http.Client{}
	req, err := http.NewRequest("GET", fmt.Sprintf("%s?ip=%s&domain=%s&token=%s", endpoint, ip, domain, token), nil)
	if err != nil {
		return "", err
	}

	// perform request
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}

	// check response status code
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("failed to update dns: %s", resp.Status)
	}

	return fmt.Sprintf("%s.%s", domain, config.DNS.Domain), nil
}

func GetAccessToken(config config.Config) (string, error) {
	token := config.DNS.Token

	// perform http request formurlencoded to the token endpoint
	authAPI := fmt.Sprintf("https://auth.devspaces.online/oauth2/token?grant_type=refresh_token&client_id=63dn4t3mq79e84m8tp1ugpt8a1&refresh_token=%s&redirect_uri=https://dns.devspaces.online/auth/callback", token)
	client := &http.Client{}
	req, err := http.NewRequest("POST", authAPI, nil)
	if err != nil {
		return "", err
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	// perform request
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}

	// check response status code
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("failed to get access token: %s", resp.Status)
	}

	// get the access token from the response
	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return "", err
	}

	accessToken := response["access_token"].(string)

	return accessToken, nil
}
