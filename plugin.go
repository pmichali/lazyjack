package lazyjack

import (
	"fmt"
	"io"
	"strings"

	"github.com/golang/glog"
)

// PluginAPI interface defines the actions for CNI plugins to implement.
type PluginAPI interface {
	WriteConfigContents(node *Node, w io.Writer) error
	Setup(n *Node) error
	Cleanup(n *Node) error
}

// BuildPodSubnetPrefixSuffix will create a pod network prefix, using the cluster
// prefix and node ID. For IPv6, if the subnet size is not a multiple of 16,
// then the node ID will be placed in the upper byte of the last part of the
// prefix. If the node ID is to be placed in the lower byte, and there is an
// upper byte, we need to make sure we pad with zero for values less than 0x10.
// The suffix will be "".
//
// For IPv4, the pod network is expected to be /24 with the node ID in the
// third octect. As a result, the third octet in the prefix is repplaced with
// the node ID. The suffix will be "0".
func BuildPodSubnetPrefix(mode, prefix string, netSize, nodeID int) (string, string) {
	if mode == "ipv4" {
		parts := strings.Split(prefix, ".")
		return fmt.Sprintf("%s.%s.%d.", parts[0], parts[1], nodeID), "0"
	}
	// IPv6...
	if (netSize % 16) != 0 {
		nodeID *= 256 // shift to upper byte
	} else if !strings.HasSuffix(prefix, ":") && nodeID < 0x10 {
		prefix += "0" // pad
	}
	return fmt.Sprintf("%s%x::", prefix, nodeID), ""
}

// WriteRange creates one IPAM range entry, based on the IP mode of the
// pod network
func WriteRange(c *Config, node *Node, i int, w io.Writer) (err error) {
	entryPrefix := `      [
        {
`
	entrySuffix := `        }
      ]%s
`
	_, err = fmt.Fprintf(w, entryPrefix)
	if err != nil {
		return err
	}
	prefix, suffix := BuildPodSubnetPrefix(c.Pod.Info[i].Mode, c.Pod.Info[i].Prefix, c.Pod.Info[i].Size, node.ID)
	_, err = fmt.Fprintf(w, "          \"subnet\": \"%s%s/%d\",\n", prefix, suffix, c.Pod.Info[i].Size)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "          \"gateway\": \"%s1\"\n", prefix)
	if err != nil {
		return err
	}
	comma := ""
	if i == 0 && c.General.Mode == DualStackNetMode {
		comma = ","
	}
	_, err = fmt.Fprintf(w, entrySuffix, comma)
	return err
}

// GenerateDefaultRoute creates the default route entry based on the
// IP mode. This will be called twice, if dual-stack mode.
func GenerateDefaultRoute(c *Config, i int) string {
	defaultRoute := "0.0.0.0/0"
	if c.Pod.Info[i].Mode == IPv6NetMode {
		defaultRoute = "::/0"
	}
	comma := ""
	if i == 0 && c.General.Mode == DualStackNetMode {
		comma = ","
	}
	return fmt.Sprintf("      {\"dst\": %q}%s\n", defaultRoute, comma)
}

// WriteConfigForIPAM creates the section of the CNI configuration that
// contains the IPAM information with subnet and gateway information for the
// pod network(s).
func WriteConfigForIPAM(c *Config, node *Node, w io.Writer) (err error) {
	header := `  "ipam": {
    "type": "host-local",
    "ranges": [
`
	rangeTrailer := `    ],
    "routes": [
`
	trailer := `    ]
  }
`
	_, err = fmt.Fprintf(w, header)
	if err != nil {
		return err
	}
	err = WriteRange(c, node, 0, w)
	if err != nil {
		return err
	}
	if c.General.Mode == DualStackNetMode {
		err = WriteRange(c, node, 1, w)
		if err != nil {
			return err
		}
	}
	_, err = fmt.Fprintf(w, rangeTrailer)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(w, GenerateDefaultRoute(c, 0))
	if err != nil {
		return err
	}
	if c.General.Mode == DualStackNetMode {
		_, err = fmt.Fprintf(w, GenerateDefaultRoute(c, 1))
		if err != nil {
			return err
		}
	}
	_, err = fmt.Fprintf(w, trailer)
	return err
}

// DoRouteOpsOnNodes builds static routes between minion and master node
// for a CNI plugin, so that pods can communicate across nodes.
func DoRouteOpsOnNodes(node *Node, c *Config, op string) error {
	if node.IsMaster || node.IsMinion {
		myID := node.ID
		for _, pInfo := range c.Pod.Info {
			if pInfo.Prefix == "" {
				continue
			}
			// User may have specified pod and management families in a different order for dual-stack
			// Select the corresponding management info to match pod info.
			mInfo := c.Mgmt.Info[0]
			if pInfo.Mode != mInfo.Mode {
				mInfo = c.Mgmt.Info[1]
			}
			for _, n := range c.Topology {
				if n.ID == myID {
					continue
				}
				if n.IsMaster || n.IsMinion {
					prefix, suffix := BuildPodSubnetPrefix(pInfo.Mode, pInfo.Prefix, pInfo.Size, n.ID)
					dest := fmt.Sprintf("%s%s/%d", prefix, suffix, pInfo.Size)
					gw := BuildGWIP(mInfo.Prefix, n.ID)
					var err error
					if op == "add" {
						err = c.General.NetMgr.AddRouteUsingInterfaceName(dest, gw, node.Interface)
						if err != nil && err.Error() == "file exists" {
							return fmt.Errorf("skipping - %s route to %s via %s as already exists", op, dest, gw)
						}
					} else {
						err = c.General.NetMgr.DeleteRouteUsingInterfaceName(dest, gw, node.Interface)
						if err != nil && err.Error() == "no such process" {
							return fmt.Errorf("skipping - %s route from %s via %s as non-existent", op, dest, gw)
						}
					}
					if err != nil {
						return fmt.Errorf("unable to %s pod network route for %s to %s: %v", op, dest, n.Name, err)
					}
					glog.V(1).Infof("Did pod network %s route for %s to %s", op, dest, n.Name)
				}
			}
		}
	}
	return nil
}

// CreateRoutesForPodNetwork establishes static routes between a node
// and all other nodes as part of the "up" operation.
func CreateRoutesForPodNetwork(node *Node, c *Config) error {
	glog.V(4).Info("Creating routes for pod network")
	return DoRouteOpsOnNodes(node, c, "add")
}

// RemoveRoutesForPodNetwork removes static routes between nodes, as
// part of the "down" operation.
func RemoveRoutesForPodNetwork(node *Node, c *Config) error {
	glog.V(4).Info("Deleting routes for pod network")
	return DoRouteOpsOnNodes(node, c, "delete")
}
