package orca

import (
	"fmt"
	"io"
	"io/ioutil"

	"github.com/golang/glog"
	"gopkg.in/yaml.v2"
)

type SupportNetwork struct {
	Subnet string `yaml:"subnet"`
	Size   int    `yaml:"size"`
	V4CIDR string `yaml:"v4cidr"`
}

type ManagementNetwork struct {
	Subnet string `yaml:"subnet"`
	Size   int    `yaml:"size"`
}

type PodNetwork struct {
	Prefix string `yaml:"prefix"`
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

type Node struct {
	Interface      string `yaml:"interface"`
	ID             int    `yaml:"id"`
	OperatingModes string `yaml:"opmodes"`
	Name           string
	IsMaster       bool
	IsMinion       bool
	IsDNS64Server  bool
	IsNAT64Server  bool
}

type Config struct {
	Plugin   string `yaml:"plugin"`
	Token    string `yaml:"token"`
	Topology map[string]Node
	Support  SupportNetwork    `yaml:"support_net"`
	Mgmt     ManagementNetwork `yaml:"mgmt_net"`
	Pod      PodNetwork        `yaml:"pod_net"`
	NAT64    NAT64Config       `yaml:"nat64"`
	DNS64    DNS64Config       `yaml:"dns64"`
}

const (
	SupportNetName = "support_net"

	DNS64BaseArea  = "/tmp/bind9"
	DNS64ConfArea  = "/tmp/bind9/conf"
	DNS64CacheArea = "/tmp/bind9/cache"
	DNS64NamedConf = "/tmp/bind9/conf/named.conf"

	KubeletSystemdArea = "/etc/systemd/system/kubelet.service.d"
	KubeletDropInFile  = "/etc/systemd/system/kubelet.service.d/20-extra-dns-args.conf"

	CNIConfArea = "/etc/cni/net.d"

	EtcHostsFile            = "/etc/hosts"
	EtcHostsBackupFile      = "/etc/hosts.bak"
	EtcResolvConfFile       = "/etc/resolv.conf"
	EtcResolvConfBackupFile = "/etc/resolv.conf.bak"
)

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
