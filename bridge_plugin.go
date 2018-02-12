package orca

import (
	"bytes"
	"fmt"

	"github.com/golang/glog"

	"io/ioutil"
)

const (
	BridgeCNIConfFile = "/etc/cni/net.d/cni.conf"
)

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

func CreateBridgeCNIConfigFile(node *Node, c *Config) error {
	contents := CreateBridgeCNIConfContents(node, c)
	err := ioutil.WriteFile(BridgeCNIConfFile, contents.Bytes(), 0755)
	if err != nil {
		return fmt.Errorf("Unable to create CNI config for bridge plugin: %s", err.Error())
	}
	return nil
}

func CreateRoutesForPodNetwork(node *Node, c *Config) error {
	glog.V(4).Info("Creating routes for pod network")
	if node.IsMaster || node.IsMinion {
		myID := node.ID
		for _, n := range c.Topology {
			if n.ID == myID {
				continue
			}
			if n.IsMaster || n.IsMinion {
				dest := BuildDestCIDR(c.Pod.Prefix, n.ID, c.Pod.Size)
				gw := BuildGWIP(c.Mgmt.Subnet, n.ID)
				err := AddRouteForPodNetwork(dest, gw, n.Interface, n.ID)
				if err != nil {
					return fmt.Errorf("Unable to add pod network route for %s to %s: %s", dest, n.Name, err.Error())
				}
				glog.V(4).Infof("Added pod network route for %s to %s", dest, n.Name)
			}
		}
	}
	glog.V(1).Info("Pod network routes created for bridge plugin")
	return nil
}

func RemoveRoutesForPodNetwork(node *Node, c *Config) error {
	glog.V(4).Info("Deleting routes for pod network")
	if node.IsMaster || node.IsMinion {
		myID := node.ID
		for _, n := range c.Topology {
			if n.ID == myID {
				continue
			}
			if n.IsMaster || n.IsMinion {
				dest := BuildDestCIDR(c.Pod.Prefix, n.ID, c.Pod.Size)
				gw := BuildGWIP(c.Mgmt.Subnet, n.ID)
				err := DeleteRouteForPodNetwork(dest, gw, n.Interface, n.ID)
				if err != nil {
					return fmt.Errorf("Unable to delete pod network route for %s to %s: %s", dest, n.Name, err.Error())
				}
				glog.V(4).Infof("Deleted pod network route for %s to %s", dest, n.Name)
			}
		}
	}
	glog.V(1).Info("Pod network routes deleted for bridge plugin")
	return nil
}
