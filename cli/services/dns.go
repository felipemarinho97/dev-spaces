package services

import (
	"fmt"
	"math/rand"

	"github.com/felipemarinho97/dev-spaces/cli/clients"
	"github.com/felipemarinho97/dev-spaces/cli/config"
)

func CreateDNSRecord(config config.Config, ip, name string) (string, error) {
	// get the domain from tag
	var domain string
	domain, err := core.GetTag(config, ip, "domain")

	domain := fmt.Sprintf("%s-%s", name, randomString(8))
	dns, err := clients.CreateDNSRecord(config, ip, domain)
	if err != nil {
		return "", err
	}

	return dns, nil
}

func randomString(n int) string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890")

	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
