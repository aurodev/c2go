package ast

type Record struct {
	Address  string
	Type     string
	Children []Node
}

func parseRecord(line string) *Record {
	groups := groupsFromRegex(
		"'(?P<type>.*)'",
		line,
	)

	return &Record{
		Address:  groups["address"],
		Type:     groups["type"],
		Children: []Node{},
	}
}

// AddChild adds a new child node. Child nodes can then be accessed with the
// Children attribute.
func (n *Record) AddChild(node Node) {
	n.Children = append(n.Children, node)
}
