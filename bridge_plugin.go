package lazyjack

import (
	"bytes"
	"fmt"

	"github.com/golang/glog"
)

// BridgePlugin implements the actions needed for the Bridge CNI plugin.
type BridgePlugin struct {
	Config *Config
}

// ConfigContents builds the CNI bridge plugin's config file
// contents. The subnet will be eight bits smaller than the pod cluster
// network size.
func (b BridgePlugin) ConfigContents(node *Node) *bytes.Buffer {
	header := `{
    "cniVersion": "0.3.1",
    "name": "bmbridge",
    "type": "bridge",
    "bridge": "br0",
    "isDefaultGateway": true,
    "ipMasq": true,
    "hairpinMode": true,
`
	middle := `    "ipam": {
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
	fmt.Fprintf(contents, "    \"mtu\": %d,\n", b.Config.Pod.MTU)
	fmt.Fprintf(contents, middle)
	prefix, suffix := BuildPodSubnetPrefix(b.Config.General.Mode, b.Config.Pod.Prefix, b.Config.Pod.Size, node.ID)
	fmt.Fprintf(contents, "              \"subnet\": \"%s%s/%d\",\n", prefix, suffix, b.Config.Pod.Size)
	fmt.Fprintf(contents, "              \"gateway\": \"%s1\"\n", prefix)
	fmt.Fprintf(contents, trailer)
	return contents
}

// Setup will take Bridge plugin specific actions to setup a node.
// Includes setting up routes between nodes.
func (b BridgePlugin) Setup(n *Node) error {
	err := CreateRoutesForPodNetwork(n, b.Config)
	if err != nil {
		// Note: May get error, if route already exists.
		return err
	}
	glog.V(4).Infof("created routes for CNI bridge plugin")
	return nil
}

// Cleanup performs Bridge plugin actions to clean up for a node. Includes
// deleting routes between nodes.
func (b BridgePlugin) Cleanup(n *Node) error {
	err := RemoveRoutesForPodNetwork(n, b.Config)
	if err != nil {
		return fmt.Errorf("unable to remove routes for bridge plugin: %v", err)
	}
	glog.V(4).Infof("removed routes for CNI bridge plugin")

	err = b.Config.General.NetMgr.RemoveBridge("br0")
	if err != nil {
		return fmt.Errorf("unable to remove br0 bridge: %v", err)
	}
	return nil
}
