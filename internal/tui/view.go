package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m *Model) View() string {
	if m.w == 0 || len(m.pools) == 0 {
		return "loading…"
	}
	if m.detail != nil {
		return m.detailView()
	}
	var out []string

	// context tabs
	var b strings.Builder
	m.at.tabX = m.at.tabX[:0]
	x := 0
	for i, t := range m.tabs {
		s := " " + t + " "
		m.at.tabX = append(m.at.tabX, x)
		x += len([]rune(s)) + 1
		if i == m.tab {
			s = accentBar.Render(s)
		}
		b.WriteString(s + dim.Render("│"))
	}
	m.at.tabX = append(m.at.tabX, x)
	b.WriteString(dim.Render(fmt.Sprintf("  data %s · auto %ds · r refresh", m.snapAt.Format("15:04:05"), int(refreshEvery.Seconds()))))
	out = append(out, b.String())
	if len(m.errs) > 0 {
		out = append(out, errS.Render(trunc(strings.Join(m.errs, " · "), m.w)))
	}

	scope := "all contexts"
	if m.tab > 0 {
		scope = m.ctxs[m.tab-1].Name
	}
	sel := m.pools[m.cur[0]]
	var hide []string
	if m.tab > 0 || len(m.ctxs) == 1 {
		hide = append(hide, "CTX")
	}
	pc := pick(poolCols, hide...)
	if !sel.All {
		hide = append(hide, "NODEPOOL")
	}
	nc := pick(nodeCols, hide...)
	m.colTitles = [2][]string{titles(pc), titles(nc)}

	footer := m.help.View(keys)
	avail := m.h - len(out) - lipgloss.Height(footer) - 5 // 2 title bars + 2 header lines + 1 blank
	rows := [2]int{min(len(m.pools), max(3, avail/3)), 0}
	rows[1] = max(1, avail-rows[0])
	m.at.rows = rows
	for t := range 2 {
		if m.cur[t] < m.off[t] {
			m.off[t] = m.cur[t]
		}
		if m.cur[t] >= m.off[t]+rows[t] {
			m.off[t] = m.cur[t] - rows[t] + 1
		}
		m.off[t] = max(0, min(m.off[t], m.count(t)-rows[t]))
	}

	out = append(out, section(m.w, m.focus == 0,
		fmt.Sprintf(" NODEPOOLS · %s · %d pools", scope, len(m.pools)-1),
		"↑↓ / click: pick a pool → its nodes below · $ = list price, no SP/RI "))
	m.at.tableY[0] = len(out)
	lines, xs := renderTable(pc, m.pools, m.sortBy[0], m.desc[0], m.cur[0], m.off[0], rows[0], m.w, m.focus == 0)
	m.at.colX[0] = xs
	out = append(out, lines...)
	out = append(out, "")

	title := fmt.Sprintf(" NODES · all pools · %d nodes · %d pods (%d pending)", len(m.nodes), m.podCount, m.podsPending)
	if !sel.All {
		where := sel.Name
		if m.tab == 0 && len(m.ctxs) > 1 {
			where = sel.Ctx + " / " + sel.Name
		}
		title = fmt.Sprintf(" NODES · %s · %d nodes · ", where, len(m.nodes))
		if sel.Spec.Policy == "MNG" {
			title += "managed node group (not Karpenter)"
		} else {
			title += fmt.Sprintf("consolidationPolicy=%s consolidateAfter=%s budget=%s", sel.Spec.Policy, sel.Spec.After, sel.Spec.Budget)
		}
	}
	out = append(out, section(m.w, m.focus == 1, title, "enter / click: pods on node "))
	m.at.tableY[1] = len(out)
	lines, xs = renderTable(nc, m.nodes, m.sortBy[1], m.desc[1], m.cur[1], m.off[1], rows[1], m.w, m.focus == 1)
	m.at.colX[1] = xs
	out = append(out, lines...)
	out = append(out, footer)
	return strings.Join(out, "\n")
}
