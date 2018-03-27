package lazyjack

import (
	"bytes"
	"fmt"
	"path/filepath"

	"github.com/golang/glog"

	"io/ioutil"
)

// CreateBridgeCNIConfContents builds the CNI bridge plugin's config file
// contents.
func CreateBridgeCNIConfContents(node *Node, c *Config) *bytes.Buffer {
	header := `{
    "cniVersion": "0.3.0",
    "name": "bmbridge",
    "type": "bridge",
    "bridge": "br0",
    "isDefaultGateway": true,
    "ipMasq": true,
    "hairpinMode": true,
    "ipam": {
        "type": "host-local",
        "ranges": [
          [
            {
`
	trailer := `	    }
          ]
        ]
    }
}
`
	contents := bytes.NewBufferString(header)
	fmt.Fprintf(contents, "              \"subnet\": \"%s:%d::/%d\",\n", c.Pod.Prefix, node.ID, c.Pod.Size)
	fmt.Fprintf(contents, "              \"gateway\": \"%s:%d::1\"\n", c.Pod.Prefix, node.ID)
	fmt.Fprintf(contents, trailer)
	return contents
}

// CreateBridgeCNIConfigFile creates the bridge plugin's configuration
// file in /etc/cni/net.d/ area.
func CreateBridgeCNIConfigFile(node *Node, c *Config) error {
	contents := CreateBridgeCNIConfContents(node, c)
	filename := filepath.Join(c.General.CNIArea, CNIConfFile)
	err := ioutil.WriteFile(filename, contents.Bytes(), 0755)
	if err != nil {
		return fmt.Errorf("unable to create CNI config for bridge plugin: %v", err)
	}
	return nil
}

// DoRouteOpsOnNodes builds static routes between minion and master node
// for the bridge plugin, so that pods can communicate across nodes.
func DoRouteOpsOnNodes(node *Node, c *Config, op string) error {
	if node.IsMaster || node.IsMinion {
		myID := node.ID
		for _, n := range c.Topology {
			if n.ID == myID {
				continue
			}
			if n.IsMaster || n.IsMinion {
				dest := BuildDestCIDR(c.Pod.Prefix, n.ID, c.Pod.Size)
				gw := BuildGWIP(c.Mgmt.Prefix, n.ID)
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
