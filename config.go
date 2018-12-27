package lazyjack

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"

	"text/template"

	"github.com/golang/glog"
	"gopkg.in/yaml.v2"
)

// NetInfo contains the prefix, size, and mode for a network.
type NetInfo struct {
	Prefix string
	Size   int
	Mode   string // "ipv6" or "ipv4"
}

// SupportNetwork defines information for the support network.
type SupportNetwork struct {
	CIDR   string  `yaml:"cidr"`
	Info   NetInfo // Internal
	V4CIDR string  `yaml:"v4cidr"`
}

// ManagementNetwork defines information for the management network.
type ManagementNetwork struct {
	CIDR  string     `yaml:"cidr"`
	CIDR2 string     `yaml:"cidr2"`
	Info  [2]NetInfo //Internal
}

// PodNetwork defines information for the the pod network.
type PodNetwork struct {
	CIDR  string     `yaml:"cidr"`
	CIDR2 string     `yaml:"cidr2"`
	Info  [2]NetInfo // Internal
	MTU   int        `yaml:"mtu"`
}

// ServiceNetwork defines information for the service network.
type ServiceNetwork struct {
	CIDR string  `yaml:"cidr"`
	Info NetInfo // Internal
}

// DNS64Config defines information for the DNS64 server configuration.
type DNS64Config struct {
	RemoteV4Server string `yaml:"remote_server"`
	CIDR           string `yaml:"cidr"`
	CIDRPrefix     string
	ServerIP       string `yaml:"ip"`
	AllowIPv6Use   bool   `yaml:"allow_ipv6_use"` // Deprecated
	AllowAAAAUse   bool   `yaml:"allow_aaaa_use"`
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
	Mode               string     `yaml:"mode"`
	Plugin             string     `yaml:"plugin"`
	Token              string     `yaml:"token"`           // Internal
	TokenCertHash      string     `yaml:"token-cert-hash"` // Internal
	WorkArea           string     `yaml:"work-area"`
	CNIPlugin          PluginAPI  // Internal
	SystemdArea        string     // Internal
	EtcArea            string     // Internal
	CNIArea            string     // Internal
	K8sCertArea        string     // Internal
	NetMgr             Networker  // Internal
	Hyper              Hypervisor // Internal
	KubeAdmVersion     string     // Internal
	FullKubeAdmVersion string     // Internal
	K8sVersion         string     `yaml:"kubernetes-version"`
	Insecure           bool       `yaml:"insecure"`
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
	// DNS64Volume name of volume holding DNS64 configuration
	DNS64Volume = "volume-bind9"
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

	// DefaultToken used when in insecure mode
	DefaultToken = "abcdef.abcdefghijklmnop"

	// MinimumPodMTU is the smallest MTU for IPv6
	MinimumPodMTU = 1280
	// DefaultPodMTU is the default MTU to use, when not specified
	DefaultPodMTU = 1500

	// IPv6NetMode for IPv6 only networks
	IPv6NetMode = "ipv6"
	// DefaultNetMode default network operating mode
	DefaultNetMode = IPv6NetMode
	// IPv4NetMode for IPv4 only network operating mode
	IPv4NetMode = "ipv4"
	// DualStackNetMode for IPv4/IPv6 network operating mode
	DualStackNetMode = "dual-stack"
)

// KubeAdmConfigInfo provides values for the templates used to populate
// the kubeadm.conf file (contents).
type KubeAdmConfigInfo struct {
	AdvertiseAddress string
	AuthToken        string
	BindAddress      string
	BindPort         int
	DNS_ServiceIP    string
	K8sVersion       string
	KubeMasterName   string
	PodNetworkCIDR   string
	ServiceSubnet    string
	UseCoreDNS       bool
}

// Template_v1_10 kubeadm.conf content template for Kubernetes V1.10
var Template_v1_10 = template.Must(template.New("v1.10").Parse(`# V1.10 (and older) based config
api:
  advertiseAddress: "{{.AdvertiseAddress}}"
apiServerExtraArgs:
  insecure-bind-address: "{{.BindAddress}}"
  insecure-port: "{{.BindPort}}"
apiVersion: kubeadm.k8s.io/v1alpha1
featureGates: {CoreDNS: {{.UseCoreDNS}}}
kind: MasterConfiguration
{{.K8sVersion}}
networking:
  # podSubnet: "{{.PodNetworkCIDR}}"
  serviceSubnet: "{{.ServiceSubnet}}"
token: "{{.AuthToken}}"
tokenTTL: 0s
nodeName: {{.KubeMasterName}}
unifiedControlPlaneImage: ""
`))

// Template_v1_11 kubeadm.conf content template for Kubernetes V1.11
var Template_v1_11 = template.Must(template.New("v1.11").Parse(`# V1.11 based config
api:
  advertiseAddress: "{{.AdvertiseAddress}}"
  bindPort: 6443
  controlPlaneEndpoint: ""
apiServerExtraArgs:
  insecure-bind-address: "{{.BindAddress}}"
  insecure-port: "{{.BindPort}}"
apiVersion: kubeadm.k8s.io/v1alpha2
auditPolicy:
  logDir: /var/log/kubernetes/audit
  logMaxAge: 2
  path: ""
bootstrapTokens:
- groups:
  - system:bootstrappers:kubeadm:default-node-token
  token: {{.AuthToken}}
  ttl: 0s
  usages:
  - signing
  - authentication
certificatesDir: /etc/kubernetes/pki
# clusterName: kubernetes
etcd:
  local:
    dataDir: /var/lib/etcd
    image: ""
featureGates: {CoreDNS: {{.UseCoreDNS}}}
kind: MasterConfiguration
kubeProxy:
  config:
    bindAddress: "{{.BindAddress}}"
    clientConnection:
      acceptContentTypes: ""
      burst: 10
      contentType: application/vnd.kubernetes.protobuf
      kubeconfig: /var/lib/kube-proxy/kubeconfig.conf
      qps: 5
    # clusterCIDR: ""
    configSyncPeriod: 15m0s
    # conntrack:
    #   max: null
    #   maxPerCore: 32768
    #   min: 131072
    #   tcpCloseWaitTimeout: 1h0m0s
    #   tcpEstablishedTimeout: 24h0m0s
    enableProfiling: false
    healthzBindAddress: 0.0.0.0:10256
    hostnameOverride: ""
    iptables:
      masqueradeAll: false
      masqueradeBit: 14
      minSyncPeriod: 0s
      syncPeriod: 30s
    ipvs:
      excludeCIDRs: null
      minSyncPeriod: 0s
      scheduler: ""
      syncPeriod: 30s
    metricsBindAddress: 127.0.0.1:10249
    mode: ""
    nodePortAddresses: null
    oomScoreAdj: -999
    portRange: ""
    resourceContainer: /kube-proxy
    udpIdleTimeout: 250ms
kubeletConfiguration:
  baseConfig:
    address: 0.0.0.0
    authentication:
      anonymous:
        enabled: false
      webhook:
        cacheTTL: 2m0s
        enabled: true
      x509:
        clientCAFile: /etc/kubernetes/pki/ca.crt
    authorization:
      mode: Webhook
      webhook:
        cacheAuthorizedTTL: 5m0s
        cacheUnauthorizedTTL: 30s
    cgroupDriver: cgroupfs
    cgroupsPerQOS: true
    clusterDNS:
    - "{{.DNS_ServiceIP}}"
    clusterDomain: cluster.local
    containerLogMaxFiles: 5
    containerLogMaxSize: 10Mi
    contentType: application/vnd.kubernetes.protobuf
    cpuCFSQuota: true
    cpuManagerPolicy: none
    cpuManagerReconcilePeriod: 10s
    enableControllerAttachDetach: true
    enableDebuggingHandlers: true
    enforceNodeAllocatable:
    - pods
    eventBurst: 10
    eventRecordQPS: 5
    evictionHard:
      imagefs.available: 15%
      memory.available: 100Mi
      nodefs.available: 10%
      nodefs.inodesFree: 5%
    evictionPressureTransitionPeriod: 5m0s
    failSwapOn: true
    fileCheckFrequency: 20s
    hairpinMode: promiscuous-bridge
    healthzBindAddress: 127.0.0.1
    healthzPort: 10248
    httpCheckFrequency: 20s
    imageGCHighThresholdPercent: 85
    imageGCLowThresholdPercent: 80
    imageMinimumGCAge: 2m0s
    iptablesDropBit: 15
    iptablesMasqueradeBit: 14
    kubeAPIBurst: 10
    kubeAPIQPS: 5
    makeIPTablesUtilChains: true
    maxOpenFiles: 1000000
    maxPods: 110
    nodeStatusUpdateFrequency: 10s
    oomScoreAdj: -999
    podPidsLimit: -1
    # port: 10250
    registryBurst: 10
    registryPullQPS: 5
    resolvConf: /etc/resolv.conf
    rotateCertificates: true
    runtimeRequestTimeout: 2m0s
    serializeImagePulls: true
    staticPodPath: /etc/kubernetes/manifests
    streamingConnectionIdleTimeout: 4h0m0s
    syncFrequency: 1m0s
    volumeStatsAggPeriod: 1m0s
{{.K8sVersion}}
networking:
  # podSubnet: "{{.PodNetworkCIDR}}"
  serviceSubnet: "{{.ServiceSubnet}}"
nodeRegistration:
  name: {{.KubeMasterName}}
unifiedControlPlaneImage: ""
`))

// Template_v1_12 kubeadm.conf content template for Kubernetes V1.12
var Template_v1_12 = template.Must(template.New("v1.12").Parse(`# V1.12 based config
apiEndpoint:
  advertiseAddress: "{{.AdvertiseAddress}}"
  bindPort: 6443
apiVersion: kubeadm.k8s.io/v1alpha3
bootstrapTokens:
- groups:
  - system:bootstrappers:kubeadm:default-node-token
  token: {{.AuthToken}}
  ttl: 0s
  usages:
  - signing
  - authentication
kind: InitConfiguration
nodeRegistration:
  criSocket: /var/run/dockershim.sock
  name: {{.KubeMasterName}}
  taints:
  - effect: NoSchedule
    key: node-role.kubernetes.io/master
---
apiServerExtraArgs:
  insecure-bind-address: "{{.BindAddress}}"
  insecure-port: "{{.BindPort}}"
apiVersion: kubeadm.k8s.io/v1alpha3
auditPolicy:
  logDir: /var/log/kubernetes/audit
  logMaxAge: 2
  path: ""
certificatesDir: /etc/kubernetes/pki
controlPlaneEndpoint: ""
etcd:
  local:
    dataDir: /var/lib/etcd
    image: ""
featureGates: {CoreDNS: {{.UseCoreDNS}}}
imageRepository: k8s.gcr.io
kind: ClusterConfiguration
{{.K8sVersion}}
networking:
  # podSubnet: "{{.PodNetworkCIDR}}"
  serviceSubnet: "{{.ServiceSubnet}}"
unifiedControlPlaneImage: ""
`))

// Template_v1_13 kubeadm.conf content template for Kubernetes V1.13
var Template_v1_13 = template.Must(template.New("v1.13").Parse(`# V1.13 based config
apiEndpoint:
  advertiseAddress: "{{.AdvertiseAddress}}"
  bindPort: 6443
apiVersion: kubeadm.k8s.io/v1beta1
bootstrapTokens:
- groups:
  - system:bootstrappers:kubeadm:default-node-token
  token: {{.AuthToken}}
  ttl: 24h0m0s
  usages:
  - signing
  - authentication
kind: InitConfiguration
nodeRegistration:
  criSocket: /var/run/dockershim.sock
  name: {{.KubeMasterName}}
  taints:
  - effect: NoSchedule
    key: node-role.kubernetes.io/master
---
apiServerExtraArgs:
  insecure-bind-address: "{{.BindAddress}}"
  insecure-port: "{{.BindPort}}"
apiVersion: kubeadm.k8s.io/v1beta1
auditPolicy:
  logDir: /var/log/kubernetes/audit
  logMaxAge: 2
  path: ""
certificatesDir: /etc/kubernetes/pki
# clusterName: kubernetes
controlPlaneEndpoint: ""
etcd:
  local:
    dataDir: /var/lib/etcd
    image: ""
featureGates: {CoreDNS: {{.UseCoreDNS}}}
imageRepository: k8s.gcr.io
kind: ClusterConfiguration
{{.K8sVersion}}
networking:
  dnsDomain: cluster.local
  # podSubnet: "{{.PodNetworkCIDR}}"
  serviceSubnet: "{{.ServiceSubnet}}"
unifiedControlPlaneImage: ""
---
apiVersion: kubeproxy.config.k8s.io/v1alpha1
bindAddress: "{{.BindAddress}}"
clientConnection:
  acceptContentTypes: ""
  burst: 10
  contentType: application/vnd.kubernetes.protobuf
  kubeconfig: /var/lib/kube-proxy/kubeconfig.conf
  qps: 5
# clusterCIDR: ""
configSyncPeriod: 15m0s
# conntrack:
#   max: null
#   maxPerCore: 32768
#   min: 131072
#   tcpCloseWaitTimeout: 1h0m0s
#   tcpEstablishedTimeout: 24h0m0s
enableProfiling: false
healthzBindAddress: 0.0.0.0:10256
hostnameOverride: ""
iptables:
  masqueradeAll: false
  masqueradeBit: 14
  minSyncPeriod: 0s
  syncPeriod: 30s
ipvs:
  excludeCIDRs: null
  minSyncPeriod: 0s
  scheduler: ""
  syncPeriod: 30s
kind: KubeProxyConfiguration
metricsBindAddress: 127.0.0.1:10249
mode: ""
nodePortAddresses: null
oomScoreAdj: -999
portRange: ""
resourceContainer: /kube-proxy
udpIdleTimeout: 250ms
---
address: 0.0.0.0
apiVersion: kubelet.config.k8s.io/v1beta1
authentication:
  anonymous:
    enabled: false
  webhook:
    cacheTTL: 2m0s
    enabled: true
  x509:
    clientCAFile: /etc/kubernetes/pki/ca.crt
authorization:
  mode: Webhook
  webhook:
    cacheAuthorizedTTL: 5m0s
    cacheUnauthorizedTTL: 30s
cgroupDriver: cgroupfs
cgroupsPerQOS: true
clusterDNS:
- "{{.DNS_ServiceIP}}"
clusterDomain: cluster.local
configMapAndSecretChangeDetectionStrategy: Watch
containerLogMaxFiles: 5
containerLogMaxSize: 10Mi
contentType: application/vnd.kubernetes.protobuf
cpuCFSQuota: true
cpuCFSQuotaPeriod: 100ms
cpuManagerPolicy: none
cpuManagerReconcilePeriod: 10s
enableControllerAttachDetach: true
enableDebuggingHandlers: true
enforceNodeAllocatable:
- pods
eventBurst: 10
eventRecordQPS: 5
evictionHard:
  imagefs.available: 15%
  memory.available: 100Mi
  nodefs.available: 10%
  nodefs.inodesFree: 5%
evictionPressureTransitionPeriod: 5m0s
failSwapOn: true
fileCheckFrequency: 20s
hairpinMode: promiscuous-bridge
healthzBindAddress: 127.0.0.1
healthzPort: 10248
httpCheckFrequency: 20s
imageGCHighThresholdPercent: 85
imageGCLowThresholdPercent: 80
imageMinimumGCAge: 2m0s
iptablesDropBit: 15
iptablesMasqueradeBit: 14
kind: KubeletConfiguration
kubeAPIBurst: 10
kubeAPIQPS: 5
makeIPTablesUtilChains: true
maxOpenFiles: 1000000
maxPods: 110
nodeLeaseDurationSeconds: 40
nodeStatusUpdateFrequency: 10s
oomScoreAdj: -999
podPidsLimit: -1
# port: 10250
registryBurst: 10
registryPullQPS: 5
resolvConf: /etc/resolv.conf
rotateCertificates: true
runtimeRequestTimeout: 2m0s
serializeImagePulls: true
staticPodPath: /etc/kubernetes/manifests
streamingConnectionIdleTimeout: 4h0m0s
syncFrequency: 1m0s
volumeStatsAggPeriod: 1m0s
`))

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

type configWriter struct {
	w   io.Writer
	buf bytes.Buffer
	err error
}

func (w *configWriter) Write(format string, a ...interface{}) {
	if w.err != nil {
		return
	}
	_, w.err = fmt.Fprintf(&w.buf, format, a...)
}

func (w *configWriter) Flush() error {
	if w.err == nil {
		_, w.err = w.buf.WriteTo(w.w)
	}
	return w.err
}
