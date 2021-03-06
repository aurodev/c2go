package ast

import (
	"testing"
)

func TestDefaultStmt(t *testing.T) {
	nodes := map[string]Node{
		`0x7f951308bfb0 <line:17:5, line:18:34>`: &DefaultStmt{
			Address:  "0x7f951308bfb0",
			Position: "line:17:5, line:18:34",
			Children: []Node{},
		},
	}

	runNodeTests(t, nodes)
}
