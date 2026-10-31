package tui

import (
	"fmt"
	"math"
	"regexp"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/open-dev-tools/karpenter-node-viewer/internal/snapshot"
)

var hostSuffix = regexp.MustCompile(`\.(ec2|[a-z0-9-]+\.compute)\.internal$`)

func consolidation(p *snapshot.Pool) cell {
	switch p.Spec.Policy {
	case "WhenEmpty": // under-used nodes are never merged
		return cell{s: "empty only", st: lipgloss.NewStyle().Foreground(cBad).Bold(true)}
	case "WhenEmptyOrUnderutilized":
		return cell{s: "underutilized", st: lipgloss.NewStyle().Foreground(cGood)}
	case "MNG":
		return cell{s: "managed NG", st: dim}
	}
	if p.All {
		return txt("")
	}
	return cell{s: p.Spec.Policy, st: dim}
}

var poolCols = []col[*snapshot.Pool]{
	{title: "CTX", cell: func(p *snapshot.Pool) cell { return txt(p.Ctx) }},
	{title: "NODEPOOL", cell: func(p *snapshot.Pool) cell {
		if p.All {
			return cell{s: p.Name, st: bold}
		}
		return txt(p.Name)
	}},
	{title: "CONSOLIDATION", cell: consolidation},
	{title: "NODES→FIT", key: func(p *snapshot.Pool) float64 { return float64(p.Nodes - p.Fit) }, cell: func(p *snapshot.Pool) cell { return num(fmt.Sprintf("%d → %d", p.Nodes, p.Fit)) }},
	{title: "CPU", key: func(p *snapshot.Pool) float64 { return p.CPUPct }, cell: func(p *snapshot.Pool) cell { return bar(p.CPUPct) }},
	{title: "MEM", key: func(p *snapshot.Pool) float64 { return p.MemPct }, cell: func(p *snapshot.Pool) cell { return bar(p.MemPct) }},
	{title: "$/MO", key: func(p *snapshot.Pool) float64 { return p.Monthly }, cell: func(p *snapshot.Pool) cell { return num(money(p.Monthly)) }},
	{title: "SAVE≈/MO", key: func(p *snapshot.Pool) float64 { return p.Save }, cell: func(p *snapshot.Pool) cell {
		c := num(money(p.Save))
		if p.Save > 0 {
			c.st = lipgloss.NewStyle().Foreground(cGood).Bold(true)
		}
		return c
	}},
	{title: "WHY NOT CONSOLIDATED", cell: func(p *snapshot.Pool) cell { return cell{s: p.Blocker, st: dim} }, flex: true},
}

var nodeCols = []col[*snapshot.Node]{
	{title: "CTX", cell: func(n *snapshot.Node) cell { return txt(n.Ctx) }},
	{title: "NODE", cell: func(n *snapshot.Node) cell { return txt(hostSuffix.ReplaceAllString(n.Name, "")) }},
	{title: "NODEPOOL", cell: func(n *snapshot.Node) cell { return txt(n.Pool) }},
	{title: "TYPE", cell: func(n *snapshot.Node) cell { return txt(n.InstanceType) }},
	{title: "CAP", cell: func(n *snapshot.Node) cell { return txt(n.Capacity) }},
	{title: "CPU", key: func(n *snapshot.Node) float64 { return n.CPUPct }, cell: func(n *snapshot.Node) cell { return bar(n.CPUPct) }},
	{title: "MEM", key: func(n *snapshot.Node) float64 { return n.MemPct }, cell: func(n *snapshot.Node) cell { return bar(n.MemPct) }},
	{title: "PODS", key: func(n *snapshot.Node) float64 { return float64(n.Pods) }, cell: func(n *snapshot.Node) cell { return num(fmt.Sprint(n.Pods)) }},
	{title: "$/H", key: func(n *snapshot.Node) float64 { return n.Price }, cell: func(n *snapshot.Node) cell {
		if math.IsNaN(n.Price) {
			return num("?")
		}
		return num(fmt.Sprintf("$%.4f", n.Price))
	}},
	{title: "IDLE/MO", key: func(n *snapshot.Node) float64 { return n.Idle }, cell: func(n *snapshot.Node) cell { return num(money(n.Idle)) }},
	{title: "AGE", key: func(n *snapshot.Node) float64 { return time.Since(n.Created).Seconds() }, cell: func(n *snapshot.Node) cell { return num(age(n.Created)) }},
	{title: "STATE", cell: func(n *snapshot.Node) cell { return cell{s: n.State, st: warn} }},
	{title: "WHY NOT CONSOLIDATED", cell: func(n *snapshot.Node) cell { return cell{s: n.Why, st: dim} }, flex: true},
}
