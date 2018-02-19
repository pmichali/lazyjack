package lazyjack_test

import (
	"github.com/pmichali/lazyjack"
	"testing"
)

func TestBridgeCNIConfigContents(t *testing.T) {
	c := &lazyjack.Config{
		Pod: lazyjack.PodNetwork{
			Prefix: "fd00:40:0:0",
			Size:   80,
		},
	}
	n := &lazyjack.Node{ID: 10}

	expected := `{
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
              "subnet": "fd00:40:0:0:10::/80",
              "gateway": "fd00:40:0:0:10::1"
	    }
          ]
        ]
    }
}
`
	actual := lazyjack.CreateBridgeCNIConfContents(n, c)
	if actual.String() != expected {
		t.Errorf("Bridge CNI config contents wrong\nExpected:\n%s\n  Actual:\n%s\n", expected, actual.String())
	}
}
