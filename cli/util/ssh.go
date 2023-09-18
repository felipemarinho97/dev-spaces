package util

import (
	"bufio"
	"fmt"
	"os"

	"github.com/felipemarinho97/dev-spaces/cli/config"
)

const (
	customSSHConfigPath = "config.d/dev-spaces"
)

func CreateSSHConfig(config config.Config, ip, name string) (string, error) {
	// get the custom ssh config path
	sshConfigPath, err := getSSHConfigPath()
	if err != nil {
		return "", err
	}

	// add entry to ssh config
	err = putConfigEntry(sshConfigPath, name, ip)
	if err != nil {
		return "", err
	}

	return name, nil
}

func getSSHConfigPath() (string, error) {
	// check if ssh default config exists, if not create it
	sshConfigPath, err := findSSHConfig()
	if err != nil {
		sshConfigPath, err = createSSHConfig()
		if err != nil {
			return "", err
		}
	}

	// check if there is a Include directive in the default config, if not add it
	sshConfig, err := os.Open(sshConfigPath)
	if err != nil {
		return "", err
	}
	defer sshConfig.Close()

	// read the file line by line
	scanner := bufio.NewScanner(sshConfig)
	for scanner.Scan() {
		line := scanner.Text()
		if line == fmt.Sprintf("Include %s/*", customSSHConfigPath) {
			// Include directive already exists
			return fmt.Sprintf("%s/.ssh/%s", os.Getenv("HOME"), customSSHConfigPath), nil
		}
	}

	// Include directive does not exist, append it
	sshConfig, err = os.OpenFile(sshConfigPath, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return "", err
	}

	_, err = sshConfig.WriteString(fmt.Sprintf("Include %s/*\n", customSSHConfigPath))
	if err != nil {
		return "", err
	}

	// create the custom ssh config directory
	err = os.MkdirAll(fmt.Sprintf("%s/.ssh/%s", os.Getenv("HOME"), customSSHConfigPath), 0700)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s/.ssh/%s", os.Getenv("HOME"), customSSHConfigPath), nil
}

func findSSHConfig() (string, error) {
	// check if ssh config exists
	sshConfigPath := fmt.Sprintf("%s/.ssh/config", os.Getenv("HOME"))
	if _, err := os.Stat(sshConfigPath); os.IsNotExist(err) {
		return "", fmt.Errorf("ssh config not found at %s", sshConfigPath)
	}

	return sshConfigPath, nil
}

func createSSHConfig() (string, error) {
	// create ssh config
	sshConfigPath := fmt.Sprintf("%s/.ssh/config", os.Getenv("HOME"))
	sshConfig, err := os.Create(sshConfigPath)
	if err != nil {
		return "", err
	}

	return sshConfigPath, sshConfig.Close()
}

func putConfigEntry(sshConfigPath, name, ip string) error {
	// create entry file
	sshConfig, err := os.Create(fmt.Sprintf("%s/%s", sshConfigPath, name))
	if err != nil {
		return err
	}

	// write the entry
	_, err = sshConfig.WriteString(fmt.Sprintf("Host %s\n\tHostName %s\n\tPort 2222\n", name, ip))
	if err != nil {
		return err
	}

	return nil
}
