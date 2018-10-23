package lazyjack

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/golang/glog"
)

// PluginAPI interface defines the actions for CNI plugins to implement.
type PluginAPI interface {
	ConfigContents(node *Node) *bytes.Buffer
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

// DoRouteOpsOnNodes builds static routes between minion and master node
// for a CNI plugin, so that pods can communicate across nodes.
func DoRouteOpsOnNodes(node *Node, c *Config, op string) error {
	if node.IsMaster || node.IsMinion {
		myID := node.ID
		for _, n := range c.Topology {
			if n.ID == myID {
				continue
			}
			if n.IsMaster || n.IsMinion {
				prefix, suffix := BuildPodSubnetPrefix(c.General.Mode, c.Pod.Info[0].Prefix, c.Pod.Info[0].Size, n.ID)
				dest := fmt.Sprintf("%s%s/%d", prefix, suffix, c.Pod.Info[0].Size)
				gw := BuildGWIP(c.Mgmt.Info[0].Prefix, n.ID)
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
