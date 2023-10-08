package util

import (
	"bufio"
	"fmt"
	"os"
	"regexp"

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

	// discard the version at the end of name (the number after the /)
	re := regexp.MustCompile(`(.*)\/.*`)
	name = re.ReplaceAllString(name, "$1")

	// add entry to ssh config
	err = putConfigEntry(sshConfigPath, name, ip)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s/%s", sshConfigPath, name), nil
}

func getSSHConfigPath() (string, error) {
	// create the custom ssh config directory
	err := os.MkdirAll(fmt.Sprintf("%s/.ssh/%s", os.Getenv("HOME"), customSSHConfigPath), 0700)
	if err != nil {
		return "", err
	}

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

	// Include directive does not exist, append it to the beginning of the file
	sshConfigContent, err := os.ReadFile(sshConfigPath)
	if err != nil {
		return "", err
	}

	newSSHConfigContent := fmt.Sprintf("Include %s/*\n%s", customSSHConfigPath, string(sshConfigContent))
	err = os.WriteFile(sshConfigPath, []byte(newSSHConfigContent), 0644)
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
	// check if entry already exists
	sshConfig, err := os.Open(fmt.Sprintf("%s/%s", sshConfigPath, name))
	// if the file does not exist, create it
	if os.IsNotExist(err) {
		// create entry file
		sshConfig, err := os.Create(fmt.Sprintf("%s/%s", sshConfigPath, name))
		if err != nil {
			return err
		}
		defer sshConfig.Close()

		// write the entry
		_, err = sshConfig.WriteString(fmt.Sprintf("Host %s\n\tHostName %s\n\tPort 2222\n\tStrictHostKeyChecking no\n\tUser root\n\t# IdentityFile <~/.ssh/your-key.pem>\n", name, ip))
		if err != nil {
			return err
		}
	}
	defer sshConfig.Close()

	// replace the HostName with the new IP address
	scanner := bufio.NewScanner(sshConfig)
	for scanner.Scan() {
		line := scanner.Text()
		re := regexp.MustCompile(`HostName\s.*`)
		if re.MatchString(line) {
			// replace the HostName
			fileContent, err := os.ReadFile(fmt.Sprintf("%s/%s", sshConfigPath, name))
			if err != nil {
				return err
			}

			newFileContent := re.ReplaceAllString(string(fileContent), fmt.Sprintf("HostName %s", ip))
			err = os.WriteFile(fmt.Sprintf("%s/%s", sshConfigPath, name), []byte(newFileContent), 0644)
			if err != nil {
				return err
			}

			return nil
		}
	}

	return nil
}
