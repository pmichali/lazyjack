package orca

import (
	"bytes"
	"fmt"
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
