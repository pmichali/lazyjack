package lazyjack

import (
	"fmt"
	"io"
	"net"
	"os"
	"regexp"
	"strings"

	"github.com/golang/glog"
	"github.com/vishvananda/netlink"
)

// ValidateCommand ensures that the command specified is supported.
func ValidateCommand(command string) (string, error) {
	if command == "" {
		return "", fmt.Errorf("missing command")
	}
	validCommands := []string{"init", "prepare", "up", "down", "clean", "version"}
	for _, c := range validCommands {
		if strings.EqualFold(c, command) {
			return c, nil
		}
	}
	return "", fmt.Errorf("unknown command %q", command)
}

// ValidateHost ensures that the host is mentioned in the configuration.
func ValidateHost(host string, config *Config) error {
	_, ok := config.Topology[host]
	if !ok {
		return fmt.Errorf("unable to find info for host %q in config file", host)
	}
	return nil
}

// ValidateUniqueIDs ensures that the node IDs are unique.
func ValidateUniqueIDs(c *Config) error {
	// Ensure no duplicate IDs
	IDs := make(map[int]string)
	for name, node := range c.Topology {
		if first, seen := IDs[node.ID]; seen {
			return fmt.Errorf("duplicate node ID %d seen for node %q and %q", node.ID, first, name)
		}
		IDs[node.ID] = name
		glog.V(4).Infof("Node %q has ID %d", name, node.ID)
	}
	return nil
}

// ValidateNodeOpModes checks that valid operational mode names are used.
// NOTE: Side effect of saving the operating modes as flags, for easier use.
func ValidateNodeOpModes(netMode string, node *Node) error {
	validModes := []string{"master", "minion"}
	if netMode == IPv6NetMode {
		validModes = append(validModes, "dns64", "nat64")
	}
	ops := strings.Split(node.OperatingModes, " ")
	anyModes := false
	for _, op := range ops {
		if op == "" {
			continue
		}
		anyModes = true
		found := false
		for _, m := range validModes {
			if strings.EqualFold(m, op) {
				found = true
				switch m {
				case "master":
					glog.V(4).Infof("Node %q configured as master", node.Name)
					node.IsMaster = true
				case "dns64":
					glog.V(4).Infof("Node %q configured as DNS64 server", node.Name)
					node.IsDNS64Server = true
				case "nat64":
					glog.V(4).Infof("Node %q configured as NAT64 server", node.Name)
					node.IsNAT64Server = true
				default:
					glog.V(4).Infof("Node %q configured as minion", node.Name)
					node.IsMinion = true
				}
			}
		}
		if !found {
			return fmt.Errorf("invalid operating mode %q for %q", op, node.Name)
		}
	}
	if !anyModes {
		return fmt.Errorf("missing operating mode for %q", node.Name)
	}
	if node.IsMaster && node.IsMinion {
		return fmt.Errorf("invalid combination of modes for %q", node.Name)
	}
	if node.IsDNS64Server && !node.IsNAT64Server {
		return fmt.Errorf("missing %q mode for %q", "nat64", node.Name)
	}
	if !node.IsDNS64Server && node.IsNAT64Server {
		return fmt.Errorf("missing %q mode for %q", "dns64", node.Name)
	}
	return nil
}

// ValidateOpModesForAllNodes checks the operation mode for all nodes,
// and ensures that there is exactly one master node. Note: Side effect
// is storing node name in node struct for ease of access
//
// TODO: determine if allow duplicate DNS/NAT nodes
// TODO: test missing DNS/NAT node
func ValidateOpModesForAllNodes(c *Config) error {
	numMasters := 0
	for name, node := range c.Topology {
		node.Name = name
		err := ValidateNodeOpModes(c.General.Mode, &node)
		if err != nil {
			return err
		}
		if node.IsMaster {
			numMasters++
		}
		if numMasters > 1 {
			return fmt.Errorf("found multiple nodes with \"master\" operating mode")
		}
		c.Topology[name] = node // Update the map with new value
	}
	if numMasters == 0 {
		return fmt.Errorf("no master node configuration")
	}

	glog.V(4).Info("All nodes have valid operating modes")
	return nil
}

// ValidateToken ensures that the token exists and seems valid. This
// check is skipped during the init operation, where the token is created.
func ValidateToken(token string, ignoreMissing bool) error {
	if token == "" {
		if ignoreMissing {
			return nil
		}
		return fmt.Errorf("missing token in config file")
	}
	if len(token) != 23 {
		return fmt.Errorf("invalid token length (%d)", len(token))
	}
	tokenRE := regexp.MustCompile("^[a-z0-9]{6}\\.[a-z0-9]{16}$")
	if tokenRE.MatchString(token) {
		return nil
	}
	return fmt.Errorf("token is invalid %q", token)
}

// ValidateTokenCertHash ensures that the token certificate hash exists
// and seems valid. This check is skipped during the init operation, where
// the hash is created.
func ValidateTokenCertHash(certHash string, ignoreMissing bool) error {
	if certHash == "" {
		if ignoreMissing {
			return nil
		}
		return fmt.Errorf("missing token certificate hash in config file")
	}
	if len(certHash) != 64 {
		return fmt.Errorf("invalid token certificate hash length (%d)", len(certHash))
	}
	hashRE := regexp.MustCompile("^[a-fA-F0-9]{64}$")
	if !hashRE.MatchString(certHash) {
		return fmt.Errorf("token certificate hash is invalid %q", certHash)
	}
	return nil
}

// ValidateCIDR ensures that the CIDR is valid.
func ValidateCIDR(which, cidr string) error {
	if cidr == "" {
		return fmt.Errorf("config missing %s CIDR", which)
	}
	_, _, err := net.ParseCIDR(cidr)
	if err != nil {
		return fmt.Errorf("unable to parse %s CIDR (%s)", which, cidr)
	}
	return nil
}

// ValidateNetworkMode makes sure that only the supported network
// modes are entered. Currently, this is ipv4 and ipv6, and the default
// is IPv6, when not specified.
func ValidateNetworkMode(c *Config) error {
	if c.General.Mode == "" {
		c.General.Mode = DefaultNetMode
	}
	c.General.Mode = strings.ToLower(c.General.Mode)
	switch c.General.Mode {
	case IPv4NetMode:
		fallthrough
	case IPv6NetMode:
		glog.Infof("Building cluster in mode %q", c.General.Mode)
	default:
		return fmt.Errorf("unsupported network mode %q entered", c.General.Mode)
	}
	return nil
}

// ValidatePlugin ensures the plugin name is valid.
// Side effect of storing legacy value into new field.
func ValidatePlugin(c *Config) error {
	// Look for legacy plugin first
	plugin := c.Plugin
	if plugin == "" {
		plugin = c.General.Plugin
	} else {
		c.General.Plugin = plugin
	}
	if plugin == "" {
		glog.Infof("No plugin specified in config file - defaulting to %q plugin", DefaultPlugin)
		c.General.Plugin = DefaultPlugin
		return nil
	}
	switch plugin {
	case "bridge":
		c.General.CNIPlugin = BridgePlugin{c}
	case "ptp":
		c.General.CNIPlugin = PointToPointPlugin{c}
	default:
		return fmt.Errorf("plugin %q not supported", plugin)
	}
	return nil
}

// GetNetAndMask obtains the network part and mask from the provided
// CIDR.
func GetNetAndMask(input string) (string, int, error) {
	_, cidr, err := net.ParseCIDR(input)
	if err != nil {
		return "", 0, err
	}
	net := cidr.IP.String()
	mask, _ := cidr.Mask.Size()
	return net, mask, nil
}

// MakePrefixFromNetwork takes the network part of the CDIR, and
// builds an expanded prefix, so that a node ID can be added later
// to form the network part of the pod network. This means expanding
// "::" as needed, so that the prefix is fully qualified (and a ::
// can be added to the end later without causing a syntax error).
// This is done by determining how many 16 bit parts are needed and
// padding each missing part with a zero.
//
// Also, if the network includes a final part that is 16 bits and
// only the upper eight bits are part of the prefix, then the lower
// byte will be removed so that the node ID can be placed there later.
//
// Lastly, if we don't have this condition of the prefix containing
// the upper eight bits of the address, we'll place a colon on the
// end.
//
// Examples:
//   fd00:40:: (72)            -> fd00:40:0:0:
//   fd00:10:20:30:4000:: (72) -> fd00:10:20:30:40
//   fd00:10:20:30:: (64)      -> fd00:10:20:30:
//   fd00:10:20:30:: (80)      -> fd00:10:20:30:0:
//
func MakePrefixFromNetwork(network string, netSize int) string {
	minPartsNeeded := netSize / 16
	parts := strings.Split(strings.TrimRight(network, ":"), ":")
	haveParts := len(parts)

	if haveParts > minPartsNeeded {
		parts[minPartsNeeded] = strings.TrimSuffix(parts[minPartsNeeded], "00")
	}
	for haveParts < minPartsNeeded {
		parts = append(parts, "0")
		haveParts++
	}
	prefix := strings.Join(parts, ":")
	if haveParts == minPartsNeeded {
		prefix += ":"
	}
	return prefix
}

// IsIPv4 takes a known good IP address and determines if it is IPv4.
func IsIPv4(ip string) bool {
	return net.ParseIP(ip).To4() != nil
}

// MakeV4PrefixFromNetwork extracts a prefix from the IPv4 address.
// It will always remove the last octet, regardless of subnet size.
func MakeV4PrefixFromNetwork(ip string) string {
	parts := strings.Split(ip, ".")
	return fmt.Sprintf("%s.%s.%s.", parts[0], parts[1], parts[2])
}

// CalculateDerivedFields splits up CIDRs into prefix and size
// for use later.
// TODO: Validate no overlaps in CIDRs
func CalculateDerivedFields(c *Config) error {
	var err error
	c.Mgmt.Prefix, c.Mgmt.Size, err = GetNetAndMask(c.Mgmt.CIDR)
	if err != nil {
		return fmt.Errorf("invalid management network CIDR: %v", err)
	}
	if IsIPv4(c.Mgmt.Prefix) {
		if c.Mgmt.Size != 8 && c.Mgmt.Size != 16 {
			return fmt.Errorf("only /8 and /16 are supported for IPv4 management network - have /%d", c.Mgmt.Size)
		}
		c.Mgmt.Prefix = MakeV4PrefixFromNetwork(c.Mgmt.Prefix)
	}

	var serviceSize int
	c.Service.Prefix, serviceSize, err = GetNetAndMask(c.Service.CIDR)
	if err != nil {
		return fmt.Errorf("invalid service network CIDR: %v", err)
	}
	if IsIPv4(c.Service.Prefix) {
		if serviceSize >= 24 {
			return fmt.Errorf("service subnet size must be /23 or larger - have /%d", serviceSize)
		}
		c.Service.Prefix = MakeV4PrefixFromNetwork(c.Service.Prefix)
		c.Service.Mode = "ipv4"
	} else {
		c.Service.Mode = "ipv6"
	}

	if c.General.Mode == IPv6NetMode {
		c.Support.Prefix, c.Support.Size, err = GetNetAndMask(c.Support.CIDR)
		if err != nil {
			return fmt.Errorf("invalid support network CIDR: %v", err)
		}
	}

	if c.Pod.CIDR != "" {
		network, netSize, err := GetNetAndMask(c.Pod.CIDR)
		if err != nil {
			return fmt.Errorf("invalid pod network CIDR: %v", err)
		}
		c.Pod.Size = netSize + 8 // Each node will get a subnet from this network
		if IsIPv4(network) {
			if netSize != 8 && netSize != 16 {
				return fmt.Errorf("only /8 and /16 are supported for IPv4 pod networks - have /%d", netSize)
			}
			c.Pod.Prefix = MakeV4PrefixFromNetwork(network)
		} else {
			c.Pod.Prefix = MakePrefixFromNetwork(network, netSize)
		}
	} else if c.Pod.Prefix == "" || c.Pod.Size == 0 {
		return fmt.Errorf("missing pod network CIDR")
	} else {
		// Legacy mode (V6 only) - will just change so that prefix has colon at end
		// NOTE: prefixes were always a multiple of 16
		if !strings.HasSuffix(c.Pod.Prefix, ":") {
			c.Pod.Prefix += ":"
		}
	}

	if c.General.Mode == IPv6NetMode {
		c.DNS64.CIDRPrefix, _, err = GetNetAndMask(c.DNS64.CIDR)
		if err != nil {
			return fmt.Errorf("invalid DNS64 CIDR: %v", err)
		}
	}

	return nil
}

// ValidatePodFields checks user supplied pod network settings, applies
// defaults, and handles any deprecated fields.
func ValidatePodFields(c *Config) error {
	if c.Pod.MTU == 0 {
		c.Pod.MTU = DefaultPodMTU
	}
	if c.Pod.MTU < MinimumPodMTU {
		return fmt.Errorf("MTU (%d) is less than minimum MTU for IPv6 (%d)", c.Pod.MTU, MinimumPodMTU)
	}
	return nil
}

// ValidateDNS64Fields checks user supplied DNS64 settings, applies
// defaults, and handles any deprecated fields.
func ValidateDNS64Fields(c *Config) error {
	if c.DNS64.AllowIPv6Use {
		c.DNS64.AllowAAAAUse = true
	}
	return nil
}

// ValidateNAT64Fields checks that the subnet for the IPv4 mapping
// address (assumed /16), contains the subnet used for the IPv4
// mapping pool, and that both are valid.
func ValidateNAT64Fields(c *Config) error {
	if c.General.Mode != IPv6NetMode {
		return nil
	}
	if c.Support.V4CIDR == "" {
		return fmt.Errorf("missing IPv4 support network CIDR")
	}
	if c.NAT64.V4MappingIP == "" {
		return fmt.Errorf("missing IPv4 mapping IP")
	}
	if c.NAT64.V4MappingCIDR == "" {
		return fmt.Errorf("missing IPv4 mapping CIDR")
	}
	_, v4SupportNet, err := net.ParseCIDR(c.Support.V4CIDR)
	if err != nil {
		return fmt.Errorf("v4 support network (%s) is invalid: %s", c.Support.V4CIDR, err.Error())
	}
	v4MappingIP := net.ParseIP(c.NAT64.V4MappingIP)
	if v4MappingIP == nil {
		return fmt.Errorf("v4 mapping IP (%s) is invalid", c.NAT64.V4MappingIP)
	}
	v4PoolIP, _, err := net.ParseCIDR(c.NAT64.V4MappingCIDR)
	if err != nil {
		return fmt.Errorf("v4 mapping CIDR (%s) is invalid: %s", c.NAT64.V4MappingCIDR, err.Error())
	}
	if !v4SupportNet.Contains(v4MappingIP) {
		return fmt.Errorf("V4 mapping IP (%s) is not within IPv4 support subnet (%s)", c.NAT64.V4MappingIP, c.Support.V4CIDR)
	}
	if !v4SupportNet.Contains(v4PoolIP) {
		return fmt.Errorf("V4 mapping CIDR (%s) is not within IPv4 support subnet (%s)", c.NAT64.V4MappingCIDR, c.Support.V4CIDR)
	}
	return nil
}

// SetupBaseAreas allows the configuration to hold the root for both
// the working files (overridable), and key configuration files. This
// will allow the user to specify a different work area in the former
// and for unit tests to specify a temp area for the latter.
func SetupBaseAreas(work, systemd, etc, cni, cert string, c *Config) {
	if c.General.WorkArea == "" {
		c.General.WorkArea = work
	}
	c.General.SystemdArea = systemd
	c.General.EtcArea = etc
	c.General.CNIArea = cni
	c.General.K8sCertArea = cert
}

// SetupHandles configures pointers to the methods that will handle
// network and hypervisor operations.
func SetupHandles(c *Config) error {
	handle, err := netlink.NewHandle()
	if err != nil {
		return fmt.Errorf("internal Error - unable to access networking package: %v", err)
	}
	c.General.NetMgr = NetMgr{Server: NetLink{h: handle}}
	c.General.Hyper = &Docker{Command: DefaultDockerCommand}
	return nil
}

// ValidateConfigContents checks contents of the config file.
// Token and certificate hash validation is ignored during init
// phase, which will generate these values. Side effect is that
// base paths are set up based on defaults (unless overriden by
// config file). The netlink library handle is set (allowing UTs
// to override and mock that library).
// TODO: Validate support net v4 subnet > NAT64 subnet
func ValidateConfigContents(c *Config, ignoreMissing bool) error {
	var err error
	if c == nil {
		return fmt.Errorf("no configuration loaded")
	}
	err = ValidatePlugin(c)
	if err != nil {
		return err
	}

	err = ValidateNetworkMode(c)
	if err != nil {
		return err
	}

	err = ValidateToken(c.General.Token, ignoreMissing)
	if err != nil {
		return err
	}
	err = ValidateTokenCertHash(c.General.TokenCertHash, ignoreMissing)
	if err != nil {
		return err
	}
	err = ValidateUniqueIDs(c)
	if err != nil {
		return err
	}
	err = ValidateOpModesForAllNodes(c)
	if err != nil {
		return err
	}

	err = ValidateCIDR("service network", c.Service.CIDR)
	if err != nil {
		return err
	}

	err = ValidatePodFields(c)
	if err != nil {
		return err
	}

	err = ValidateDNS64Fields(c)
	if err != nil {
		return err
	}

	err = ValidateNAT64Fields(c)
	if err != nil {
		return err
	}

	err = CalculateDerivedFields(c)
	if err != nil {
		return err
	}

	err = SetupHandles(c)
	if err != nil {
		return err
	}

	SetupBaseAreas(WorkArea, KubeletSystemdArea, EtcArea, CNIConfArea, KubernetesCertArea, c)

	// FUTURE: Check no overlapping management/support/pod networks, validate IPs
	glog.V(1).Info("Configuration is valid")
	return nil
}

// LoadConfig parses the stream provided into the configuration structure.
func LoadConfig(cf io.ReadCloser) (*Config, error) {
	defer cf.Close()

	config, err := ParseConfig(cf)
	if err != nil {
		return nil, err
	}

	glog.V(1).Info("Configuration loaded")
	return config, nil
}

// OpenConfigFile opens the TAML file with configuration settings.
func OpenConfigFile(configFile string) (io.ReadCloser, error) {
	glog.V(1).Infof("Reading configuration file %q", configFile)

	cf, err := os.Open(configFile)
	if err != nil {
		return nil, fmt.Errorf("unable to open config file %q: %v", configFile, err)
	}
	return cf, nil
}
