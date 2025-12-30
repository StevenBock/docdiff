package graph

import (
	"bytes"
	"fmt"
	"sort"
)

type DOTFormatter struct {
	RankDir string
}

func (d *DOTFormatter) Format(g *Graph) ([]byte, error) {
	var buf bytes.Buffer

	rankDir := d.RankDir
	if rankDir == "" {
		rankDir = "LR"
	}

	buf.WriteString("digraph docdiff {\n")
	buf.WriteString(fmt.Sprintf("    rankdir=%s;\n", rankDir))
	buf.WriteString("    node [fontname=\"Helvetica\"];\n")
	buf.WriteString("\n")

	docNodes, sourceNodes := d.partitionNodes(g)

	if len(docNodes) > 0 {
		buf.WriteString("    // Documentation nodes\n")
		for _, node := range docNodes {
			d.writeDocNode(&buf, node)
		}
		buf.WriteString("\n")
	}

	if len(sourceNodes) > 0 {
		buf.WriteString("    // Source file nodes\n")
		for _, node := range sourceNodes {
			d.writeSourceNode(&buf, node)
		}
		buf.WriteString("\n")
	}

	if len(g.Edges) > 0 {
		buf.WriteString("    // Relationships\n")
		for _, edge := range g.Edges {
			d.writeEdge(&buf, edge)
		}
	}

	buf.WriteString("}\n")

	return buf.Bytes(), nil
}

func (d *DOTFormatter) partitionNodes(g *Graph) ([]*Node, []*Node) {
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

func (d *DOTFormatter) writeDocNode(buf *bytes.Buffer, node *Node) {
	if node.IsStale {
		buf.WriteString(fmt.Sprintf("    %s [label=\"%s\" shape=note style=filled fillcolor=\"#ffcccc\" color=\"#cc0000\"];\n",
			node.ID, node.Label))
	} else {
		buf.WriteString(fmt.Sprintf("    %s [label=\"%s\" shape=note];\n",
			node.ID, node.Label))
	}
}

func (d *DOTFormatter) writeSourceNode(buf *bytes.Buffer, node *Node) {
	buf.WriteString(fmt.Sprintf("    %s [label=\"%s\" shape=box];\n",
		node.ID, node.Label))
}

func (d *DOTFormatter) writeEdge(buf *bytes.Buffer, edge *Edge) {
	if edge.IsStale {
		buf.WriteString(fmt.Sprintf("    %s -> %s [style=dashed color=\"#cc0000\"];\n",
			edge.From, edge.To))
	} else {
		buf.WriteString(fmt.Sprintf("    %s -> %s;\n",
			edge.From, edge.To))
	}
}
