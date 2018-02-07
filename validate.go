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

func contains(ids []int, id int) bool {
	for _, i := range ids {
		if i == id {
			return true
		}
	}
	return false
}

func ValidateUniqueIDs(c *Config) error {
	// Ensure no duplicate IDs
	IDs := make(map[int]string)
	for name, node := range c.Topology {
		if first, seen := IDs[node.ID]; seen {
			return fmt.Errorf("Duplicate node ID %d seen for node %q and %q", node.ID, first, name)
		}
		IDs[node.ID] = name
	}
	return nil
}

func ValidateNodeOpModes(opModes, node string) error {
	if opModes == "" {
		return fmt.Errorf("Missing operating mode for %q", node)
	}
	validModes := []string{"master", "minion", "dns64", "nat64"}
	modesSeen := map[string]bool{"master": false, "minion": false, "dns64": false, "nat64": false}
	
	ops := strings.Split(opModes, " ")
	for _, op := range ops {
		found := false
		for _, m := range validModes {
			if strings.EqualFold(m, op) {
				found = true
				modesSeen[m] = true
			}
		}
		if ! found {
			return fmt.Errorf("Invalid operating mode %q for %q", op, node)
		}
	}
	if modesSeen["minion"] && modesSeen["master"] {
		return fmt.Errorf("Invalid combination of modes for %q", node)
	}
	if modesSeen["dns64"] && !modesSeen["nat64"] {
		return fmt.Errorf("Missing %q mode for %q", "nat64", node)
	}
	if ! modesSeen["dns64"] && modesSeen["nat64"] {
		return fmt.Errorf("Missing %q mode for %q", "dns64", node)
	}
	return nil
}

// TODO: test duplicate masters, determine if allow duplicate DNS/NAT nodes, missing DNS/NAT node
// TODO: Store mode flags as side effect?
func ValidateOpModesForAllNodes(c *Config) error {
	for name, node := range c.Topology {
		err := ValidateNodeOpModes(node.OperatingModes, name)
		if err != nil {
			return err
		}
	}
	return nil
}

func ValidateConfigContents(c *Config) error {
	if c == nil {
		return fmt.Errorf("No configuration loaded")
	}
	err := ValidateUniqueIDs(c)
	if err != nil {
		return err
	}
	err = ValidateOpModesForAllNodes(c)
	if err != nil {
		return err
	}
	
	// FUTURE: Check no overlapping management/support/pod networks
	return nil
}

func LoadConfig(cf io.ReadCloser) (*Config, error) {
	defer cf.Close()

	config, err := ParseConfig(cf)
	if err != nil {
		return nil, err
	}

	return config, nil
}

func ValidateConfigFile(configFile string) (io.ReadCloser, error) {
	glog.V(1).Infof("Using config %q", configFile)

	cf, err := os.Open(configFile)
	if err != nil {
		return nil, fmt.Errorf("Unable to open config file %q: %s", configFile, err.Error())
	}
	return cf, nil
}
