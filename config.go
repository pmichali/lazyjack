package lazyjack

import (
	"fmt"
	"io"
	"io/ioutil"

	"github.com/golang/glog"
	"gopkg.in/yaml.v2"
)

type SupportNetwork struct {
	CIDR   string `yaml:"cidr"`
	Prefix string
	Size   int
	V4CIDR string `yaml:"v4cidr"`
}

type ManagementNetwork struct {
	CIDR   string `yaml:"cidr"`
	Prefix string
	Size   int
}

type PodNetwork struct {
	Prefix string `yaml:"prefix"`
	Size   int    `yaml:"size"`
}

type ServiceNetwork struct {
	CIDR string `yaml:"cidr"`
}

type DNS64Config struct {
	RemoteV4Server string `yaml:"remote_server"`
	CIDR           string `yaml:"cidr"`
	CIDRPrefix     string
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

type GeneralSettings struct {
	Plugin        string `yaml:"plugin"`
	Token         string `yaml:"token"`           // Internal
	TokenCertHash string `yaml:"token-cert-hash"` // Internal
	WorkArea      string // Internal
	SystemdArea   string // Internal
	EtcArea       string // Internal
}

type Config struct {
	Plugin   string          `yaml:"plugin"` // Deprecated
	General  GeneralSettings `yaml:"general"`
	Topology map[string]Node
	Support  SupportNetwork    `yaml:"support_net"`
	Mgmt     ManagementNetwork `yaml:"mgmt_net"`
	Pod      PodNetwork        `yaml:"pod_net"`
	Service  ServiceNetwork    `yaml:"service_net"`
	NAT64    NAT64Config       `yaml:"nat64"`
	DNS64    DNS64Config       `yaml:"dns64"`
}

const (
	DefaultPlugin  = "bridge"
	SupportNetName = "support_net"

	WorkArea           = "/tmp/lazyjack"
	CertArea           = "certs"
	EtcArea            = "/etc"
	KubernetesCertArea = "/etc/kubernetes/pki"

	DNS64BaseArea  = "/tmp/lazyjack/bind9"
	DNS64ConfArea  = "/tmp/lazyjack/bind9/conf"
	DNS64CacheArea = "/tmp/lazyjack/bind9/cache"
	DNS64NamedConf = "/tmp/lazyjack/bind9/conf/named.conf"

	KubeletSystemdArea = "/etc/systemd/system/kubelet.service.d"
	KubeletDropInFile  = "20-extra-dns-args.conf"

	CNIConfArea = "/etc/cni/net.d"

	EtcHostsFile            = "hosts"
	EtcHostsBackupFile      = "hosts.bak"
	EtcResolvConfFile       = "resolv.conf"
	EtcResolvConfBackupFile = "resolv.conf.bak"

	KubeAdmConfFile = "kubeadm.conf"
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
