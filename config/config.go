package config

import (
	"fmt"
	"os"

	"github.com/felipemarinho97/dev-spaces/util"
	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/toml"
	"github.com/knadh/koanf/providers/file"
)

type Config struct {
	// DefaultRegion is the default AWS region to use when making AWS API calls.
	DefaultRegion string `koanf:"default_region"`
	DNS           struct {
		// Endpoint is the endpoint to use for the DNS provider.
		Endpoint string `koanf:"endpoint"`
		// Token is the token to use for the DNS provider.
		Token string `koanf:"token"`
		// Domain is the domain to use for SLD.
		Domain string `koanf:"domain"`
	} `koanf:"dynamicdns"`
}

var (
	k                 = koanf.New(".")
	AppConfig *Config = &Config{}
)

func LoadConfig() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	files := []string{
		fmt.Sprintf("%s/.config/dev-spaces/config.toml", home),
		fmt.Sprintf("%s/.dev-spaces/config.toml", home),
		"/etc/opt/dev-spaces/config.toml",
		"/etc/dev-spaces/config.toml",
		"config.toml",
	}

	tomlParser := toml.Parser()

	for _, _file := range files {
		if _, err := os.Stat(_file); err == nil {
			err := k.Load(file.Provider(_file), tomlParser)
			if err != nil {
				return err
			}
			break
		}
	}

	err = k.Unmarshal("", AppConfig)
	if err != nil {
		return err
	}

	err = util.Validator.Struct(*AppConfig)
	if err != nil {
		return err
	}

	return nil
}
