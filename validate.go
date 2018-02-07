package orca

import (
	"fmt"
	"github.com/golang/glog"
	"io"
	"os"
	"strings"
)

func ValidateCommand(command string) (string, error) {
	if command == "" {
		return "", fmt.Errorf("Missing command")
	}
	validCommands := []string{"prepare", "up", "down", "clean"}
	for _, c := range validCommands {
		if strings.EqualFold(c, command) {
			return c, nil
		}
	}
	return "", fmt.Errorf("Unknown command %q", command)
}

func ValidateHost(host string, config *Config) (*Node, error) {
	nodeInfo, ok := config.Topology[host]
	if !ok {
		return nil, fmt.Errorf("Unable to find info for host %q in config file\n", host)
	}
	return &nodeInfo, nil
}

func LoadConfig(cf io.Reader) (*Config, error) {
	config, err := ParseConfig(cf)
	if err != nil {
		return nil, fmt.Errorf("Unable to parse config file %q: %s", cf, err.Error())
	}

	return config, nil
}

func ValidateAndLoadConfig(configFile string) (*Config, error) {
	glog.V(1).Infof("Using config %q", configFile)

	cf, err := os.Open(configFile)
	if err != nil {
		return nil, fmt.Errorf("Unable to open config file %q: %s", configFile, err.Error())
	}
	defer cf.Close()
	return LoadConfig(cf)
}
