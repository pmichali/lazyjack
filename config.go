package lazyjack

import (
	"fmt"
	"io"
	"io/ioutil"

	"github.com/golang/glog"
	"gopkg.in/yaml.v2"
)

// SupportNetwork defines information for the support network.
type SupportNetwork struct {
	CIDR   string `yaml:"cidr"`
	Prefix string
	Size   int
	V4CIDR string `yaml:"v4cidr"`
}

// ManagementNetwork defines information for the management network.
type ManagementNetwork struct {
	CIDR   string `yaml:"cidr"`
	Prefix string
	Size   int
}

// PodNetwork defines information for the the pod network.
type PodNetwork struct {
	CIDR   string `yaml:"cidr"`
	Prefix string `yaml:"prefix"` // For backward compatibility
	Size   int    `yaml:"size"`   // For backward compatibility
}

// ServiceNetwork defines information for the service network.
type ServiceNetwork struct {
	CIDR string `yaml:"cidr"`
}

// DNS64Config defines information for the DNS64 server configuration.
type DNS64Config struct {
	RemoteV4Server string `yaml:"remote_server"`
	CIDR           string `yaml:"cidr"`
	CIDRPrefix     string
	ServerIP       string `yaml:"ip"`
	AllowIPv6Use   bool   `yaml:"allow_ipv6_use"`
}

// NAT64Config defines information for the NAT64 server configuration.
type NAT64Config struct {
	V4MappingCIDR string `yaml:"v4_cidr"`
	V4MappingIP   string `yaml:"v4_ip"`
	ServerIP      string `yaml:"ip"`
}

// Node defines information for the node.
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

// GeneralSettings defines general settings used by the app.
type GeneralSettings struct {
	Plugin        string     `yaml:"plugin"`
	Token         string     `yaml:"token"`           // Internal
	TokenCertHash string     `yaml:"token-cert-hash"` // Internal
	WorkArea      string     `yaml:"work-area"`
	SystemdArea   string     // Internal
	EtcArea       string     // Internal
	CNIArea       string     // Internal
	K8sCertArea   string     // Internal
	NetMgr        Networker  // Internal
	Hyper         Hypervisor // Internal
}

// Config defines the top level configuration read from YAML file.
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
	// DefaultPlugin if none is specified
	DefaultPlugin = "bridge"
	// SupportNetName used by NAT64/DNS64 server
	SupportNetName = "support_net"

	// ResourceNotPresent status, indicating it can be created
	ResourceNotPresent = "not-present"
	// ResourceRunning status, indicating it is already created
	ResourceRunning = "running"
	// ResourceExists status, indicating that it exists, but is not running
	ResourceExists = "exists"

	// WorkArea where configuration files are placed for running program
	WorkArea = "/tmp/lazyjack"
	// CertArea where certificates and keys are stored
	CertArea = "certs"
	// KubernetesCertArea where KubeAdm references certificates and keys
	KubernetesCertArea = "/etc/kubernetes/pki"

	// DNS64Name name of the DNS64 server
	DNS64Name = "bind9"
	// DNS64BaseArea root of where DNS64 directory tree exists
	DNS64BaseArea = "bind9"
	// DNS64ConfArea subdirectory for config files
	DNS64ConfArea = "conf"
	// DNS64CacheArea subdirectory for runtime cache files
	DNS64CacheArea = "cache"
	// DNS64NamedConf main configuration file
	DNS64NamedConf = "named.conf"

	// NAT64Name name of NAT64 server
	NAT64Name = "tayga"

	// KubeletSystemdArea where kubelet configuration files are located
	KubeletSystemdArea = "/etc/systemd/system/kubelet.service.d"
	// KubeletDropInFile name of drop-in file being created
	KubeletDropInFile = "20-extra-dns-args.conf"

	// CNIConfArea where CNI config files are stored
	CNIConfArea = "/etc/cni/net.d"
	// CNIConfFile name of the CNI config file
	CNIConfFile = "cni.conf"

	// EtcArea top level area for config files
	EtcArea = "/etc"
	// EtcHostsFile name of the hosts file
	EtcHostsFile = "hosts"
	// EtcHostsBackupFile backup name for hosts file
	EtcHostsBackupFile = "hosts.bak"
	// EtcResolvConfFile name of the nameserver file
	EtcResolvConfFile = "resolv.conf"
	// EtcResolvConfBackupFile backup name for nameserver file
	EtcResolvConfBackupFile = "resolv.conf.bak"

	// KubeAdmConfFile name of the configuration file used by KubeAdm
	KubeAdmConfFile = "kubeadm.conf"
)

// ParseConfig parses the YAML configuration provided, into the config structure.
func ParseConfig(configReader io.Reader) (*Config, error) {
	var config Config

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
