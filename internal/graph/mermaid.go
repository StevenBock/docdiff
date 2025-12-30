package graph

import (
	"bytes"
	"fmt"
	"sort"
)

type MermaidFormatter struct {
	Direction string
}

func (m *MermaidFormatter) Format(g *Graph) ([]byte, error) {
	var buf bytes.Buffer

	direction := m.Direction
	if direction == "" {
		direction = "LR"
	}

	buf.WriteString(fmt.Sprintf("graph %s\n", direction))

	docNodes, sourceNodes := m.partitionNodes(g)

	if len(docNodes) > 0 {
		buf.WriteString("    %% Documentation nodes\n")
		for _, node := range docNodes {
			m.writeDocNode(&buf, node)
		}
	}

	if len(sourceNodes) > 0 {
		buf.WriteString("    %% Source file nodes\n")
		for _, node := range sourceNodes {
			m.writeSourceNode(&buf, node)
		}
	}

	if len(g.Edges) > 0 {
		buf.WriteString("    %% Relationships\n")
		for i, edge := range g.Edges {
			m.writeEdge(&buf, edge, i)
		}
	}

	staleDocIDs, staleEdgeIndices := m.collectStaleElements(g)

	if len(staleDocIDs) > 0 || len(staleEdgeIndices) > 0 {
		buf.WriteString("    %% Stale styling\n")
		for _, id := range staleDocIDs {
			buf.WriteString(fmt.Sprintf("    style %s fill:#ffcccc,stroke:#cc0000\n", id))
		}
		for _, idx := range staleEdgeIndices {
			buf.WriteString(fmt.Sprintf("    linkStyle %d stroke:#cc0000\n", idx))
		}
	}

	return buf.Bytes(), nil
}

func (m *MermaidFormatter) partitionNodes(g *Graph) ([]*Node, []*Node) {
	var docNodes, sourceNodes []*Node

	for _, node := range g.Nodes {
		if node.Type == NodeTypeDoc {
			docNodes = append(docNodes, node)
		} else {
			sourceNodes = append(sourceNodes, node)
		}
	}

	sort.Slice(docNodes, func(i, j int) bool {
		return docNodes[i].ID < docNodes[j].ID
	})
	sort.Slice(sourceNodes, func(i, j int) bool {
		return sourceNodes[i].ID < sourceNodes[j].ID
	})

	return docNodes, sourceNodes
}

func (m *MermaidFormatter) writeDocNode(buf *bytes.Buffer, node *Node) {
	buf.WriteString(fmt.Sprintf("    %s[[\"%s\"]]\n", node.ID, node.Label))
}

func (m *MermaidFormatter) writeSourceNode(buf *bytes.Buffer, node *Node) {
	buf.WriteString(fmt.Sprintf("    %s(\"%s\")\n", node.ID, node.Label))
}

func (m *MermaidFormatter) writeEdge(buf *bytes.Buffer, edge *Edge, _ int) {
	if edge.IsStale {
		buf.WriteString(fmt.Sprintf("    %s -.-> %s\n", edge.From, edge.To))
	} else {
		buf.WriteString(fmt.Sprintf("    %s --> %s\n", edge.From, edge.To))
	}
}

func (m *MermaidFormatter) collectStaleElements(g *Graph) ([]string, []int) {
	var staleDocIDs []string
	var staleEdgeIndices []int

	docNodes, _ := m.partitionNodes(g)
	for _, node := range docNodes {
		if node.IsStale {
			staleDocIDs = append(staleDocIDs, node.ID)
		}
	}

	for i, edge := range g.Edges {
		if edge.IsStale {
			staleEdgeIndices = append(staleEdgeIndices, i)
		}
	}

	return staleDocIDs, staleEdgeIndices
}
