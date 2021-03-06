package ast

import (
	"testing"
)

func TestPackedAttr(t *testing.T) {
	nodes := map[string]Node{
		`0x7fae33b1ed40 <line:551:18>`: &PackedAttr{
			Address:  "0x7fae33b1ed40",
			Position: "line:551:18",
			Children: []Node{},
		},
	}

	runNodeTests(t, nodes)
}
