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

func ValidateHost(host string, config *Config) error {
	_, ok := config.Topology[host]
	if !ok {
		return fmt.Errorf("unable to find info for host %q in config file", host)
	}
	return nil
}

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
func ValidateNodeOpModes(node *Node) error {
	validModes := []string{"master", "minion", "dns64", "nat64"}

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
		err := ValidateNodeOpModes(&node)
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
	if plugin != "bridge" {
		return fmt.Errorf("plugin %q not supported", plugin)
	}
	return nil
}

func GetNetAndMask(input string) (string, int, error) {
	_, cidr, err := net.ParseCIDR(input)
	if err != nil {
		return "", 0, err
	}
	net := cidr.IP.String()
	mask, _ := cidr.Mask.Size()
	return net, mask, nil
}

// CalculateDerivedFields splits up CIDRs into prefix and size
// for use later.
// TODO: Validate no overlaps in CIDRs
func CalculateDerivedFields(c *Config) error {
	// Calculate derived fields
	var err error
	c.Mgmt.Prefix, c.Mgmt.Size, err = GetNetAndMask(c.Mgmt.CIDR)
	if err != nil {
		return fmt.Errorf("invalid management network CIDR: %v", err)
	}

	c.Support.Prefix, c.Support.Size, err = GetNetAndMask(c.Support.CIDR)
	if err != nil {
		return fmt.Errorf("invalid support network CIDR: %v", err)
	}

	c.DNS64.CIDRPrefix, _, err = GetNetAndMask(c.DNS64.CIDR)
	if err != nil {
		return fmt.Errorf("invalid DNS64 CIDR: %v", err)
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

func SetupHandles(c *Config) error {
	handle, err := netlink.NewHandle()
	if err != nil {
		return fmt.Errorf("internal Error - unable to access networking package: %v", err)
	}
	c.General.NetMgr = NetMgr{Server: NetLink{h: handle}}
	c.General.Hyper = &Docker{}
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
	if c == nil {
		return fmt.Errorf("no configuration loaded")
	}
	err := ValidateToken(c.General.Token, ignoreMissing)
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

	err = ValidatePlugin(c)
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

func LoadConfig(cf io.ReadCloser) (*Config, error) {
	defer cf.Close()

	config, err := ParseConfig(cf)
	if err != nil {
		return nil, err
	}

	glog.V(1).Info("Configuration loaded")
	return config, nil
}

func OpenConfigFile(configFile string) (io.ReadCloser, error) {
	glog.V(1).Infof("Reading configuration file %q", configFile)

	cf, err := os.Open(configFile)
	if err != nil {
		return nil, fmt.Errorf("unable to open config file %q: %v", configFile, err)
	}
	return cf, nil
}
