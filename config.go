package orca

import (
	"fmt"
	"github.com/golang/glog"
	"gopkg.in/yaml.v2"
	"io"
	"io/ioutil"
)

type SupportNetwork struct {
	Subnet string `yaml:"subnet"`
	Size   int    `yaml:"size"`
}

type ManagementNetwork struct {
	Subnet string `yaml:"subnet"`
	Size   int    `yaml:"size"`
}

type DNS64Config struct {
	RemoteV4Server string `yaml:"remote_server"`
	Prefix         string `yaml:"prefix"`
	PrefixSize     int    `yaml:"prefix_size"`
	ServerIP       string `yaml:"ip"`
}

type NAT64Config struct {
	V4MappingCIDR string `yaml:"v4_cidr"`
	V4MappingIP   string `yaml:"v4_ip"`
	ServerIP      string `yaml:"ip"`
}

const (
	MasterMode = "master"
	MinionMode = "minion"
	DNS64Mode  = "dns64"
	NAT64Mode  = "nat64"
)

type Node struct {
	Interface      string `yaml:"interface"`
	ID             int    `yaml:"id"`
	OperatingModes string `yaml:"opmodes"`
}

type Config struct {
	Plugin   string `yaml:"plugin"`
	Topology map[string]Node
	Support  SupportNetwork    `yaml:"support_net"`
	Mgmt     ManagementNetwork `yaml:"mgmt_net"`
	NAT64    NAT64Config       `yaml:"nat64"`
	DNS64    DNS64Config       `yaml:"dns64"`
}

func ParseConfig(configReader io.Reader) (*Config, error) {
	var config Config

	if configReader == nil {
		return nil, fmt.Errorf("Missing configuration file")
	}
	configContents, err := ioutil.ReadAll(configReader)
	if err != nil {
		return nil, fmt.Errorf("Failed to read config: %s", err.Error())
	}
	err = yaml.Unmarshal(configContents, &config)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse config: %s", err.Error())
	}
	glog.V(4).Infof("Configuration read %+v", config)
	return &config, nil
}
