package tui

import (
	"fmt"
	"math"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/open-dev-tools/karpenter-node-viewer/internal/snapshot"
)

// podsView lists the pods on one node, largest CPU request first.
type podsView struct {
	node *snapshot.Node
	rows []snapshot.Pod
	off  int
}

var podCols = []col[snapshot.Pod]{
	{title: "NAMESPACE", cell: func(p snapshot.Pod) cell { return txt(p.Namespace) }},
	{title: "POD", cell: func(p snapshot.Pod) cell { return txt(p.Name) }, flex: true},
	{title: "CPU", key: func(p snapshot.Pod) float64 { return p.CPU }, cell: func(p snapshot.Pod) cell { return num(fmt.Sprintf("%.3f", p.CPU)) }},
	{title: "MEM Gi", key: func(p snapshot.Pod) float64 { return p.Mem }, cell: func(p snapshot.Pod) cell { return num(fmt.Sprintf("%.2f", p.Mem/(1<<30))) }},
}

func (m *Model) openPods() {
	if len(m.nodes) == 0 {
		return
	}
	n := m.nodes[m.cur[1]]
	rows := n.PodList()
	sortRows(podCols, rows, "CPU", true)
	m.detail = &podsView{node: n, rows: rows}
}

func (m *Model) onDetailKey(msg tea.KeyMsg) (quit bool) {
	switch {
	case msg.String() == "ctrl+c":
		return true
	case key.Matches(msg, keys.Esc, keys.Quit, keys.Enter):
		m.detail = nil
	case key.Matches(msg, keys.Up):
		m.detail.off--
	case key.Matches(msg, keys.Down):
		m.detail.off++
	}
	return false
}

func (m *Model) detailView() string {
	d, n := m.detail, m.detail.node
	price := "?"
	if !math.IsNaN(n.Price) {
		price = fmt.Sprintf("$%.4f/h", n.Price)
	}
	out := []string{
		section(m.w, true, fmt.Sprintf(" PODS · %s / %s · %s · %s · %s · %s %s", n.Ctx, n.Name, n.Pool, n.InstanceType, n.Capacity, price, n.State), "esc: close "),
		fmt.Sprintf("  CPU %.2f / %.2f (%.0f%%)   MEM %.1f / %.1f Gi (%.0f%%)   pods %d",
			n.CPUReq, n.CPUAlloc, n.CPUPct*100, n.MemReq/(1<<30), n.MemAlloc/(1<<30), n.MemPct*100, len(d.rows)),
	}
	if n.Why != "" {
		out = append(out, warn.Render("  why not consolidated: "+n.Why))
	}
	out = append(out, "")
	footer := m.help.View(detailKeys)
	height := max(1, m.h-len(out)-lipgloss.Height(footer)-1)
	d.off = max(0, min(d.off, len(d.rows)-height))
	lines, _ := renderTable(podCols, d.rows, "CPU", true, -1, d.off, height, m.w, false)
	out = append(out, lines...)
	return strings.Join(append(out, footer), "\n")
}
