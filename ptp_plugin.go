package lazyjack

import (
	"bytes"
	"fmt"

	"github.com/golang/glog"
)

// PointToPointPlugin implements the actions needed for the PTP CNI plugin.
type PointToPointPlugin struct {
	Config *Config
}

// ConfigContents builds the CNI PTP plugin's config file
// contents.
func (p PointToPointPlugin) ConfigContents(node *Node) *bytes.Buffer {
	// TODO
	header := `{
  "cniVersion": "0.3.1",
  "name": "dindnet",
  "type": "ptp",
  "ipMasq": true,
`
	middle := `  "ipam": {
    "type": "host-local",
`
	trailer := `    "routes": [
      {"dst": "::/0"}
    ]
  }
}
`
	contents := bytes.NewBufferString(header)
	fmt.Fprintf(contents, "  \"mtu\": %d,\n", p.Config.Pod.MTU)
	fmt.Fprintf(contents, middle)
	prefix := BuildPodSubnetPrefix(p.Config.Pod.Prefix, p.Config.Pod.Size, node.ID)
	fmt.Fprintf(contents, "    \"subnet\": \"%s/%d\",\n", prefix, p.Config.Pod.Size)
	fmt.Fprintf(contents, trailer)
	return contents
}

// Setup will take PTP plugin specific actions to setup a node.
// Includes setting up routes between nodes.
func (p PointToPointPlugin) Setup(n *Node) error {
	err := CreateRoutesForPodNetwork(n, p.Config)
	if err != nil {
		// Note: May get error, if route already exists.
		return err
	}
	glog.V(4).Infof("created routes for CNI PTP plugin")
	return nil
}

// Cleanup performs PTP plugin actions to clean up for a node. Includes
// deleting routes between nodes.
func (p PointToPointPlugin) Cleanup(n *Node) error {
	err := RemoveRoutesForPodNetwork(n, p.Config)
	if err != nil {
		return fmt.Errorf("unable to remove routes for PTP plugin: %v", err)
	}
	glog.V(4).Infof("removed routes for CNI PTP plugin")
	return nil
}
