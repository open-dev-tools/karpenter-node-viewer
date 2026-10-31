// Package tui draws the NodePool and node tables and handles input.
package tui

import (
	"fmt"
	"slices"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/open-dev-tools/karpenter-node-viewer/internal/cluster"
	"github.com/open-dev-tools/karpenter-node-viewer/internal/snapshot"
)

const refreshEvery = 30 * time.Second

// Model is the bubbletea model. Cluster data is re-read only in snapshot (on a timer or "r");
// cursor, sort, tab and pool changes re-derive the tables from the last snapshot.
type Model struct {
	ctxs []*cluster.Context
	tab  int // 0 = all contexts, i = ctxs[i-1]

	focus     int // 0 pools, 1 nodes
	cur, off  [2]int
	selKey    [2]string // keeps the cursor on the same row across re-sorts
	sortBy    [2]string
	desc      [2]bool
	shownPool string // pool shown in the node table; the node cursor resets when it changes

	snap             []snapshot.Context
	started          time.Time
	snapAt, nextSnap time.Time

	pools                 []*snapshot.Pool // [0] is the "(all pools)" row
	nodes                 []*snapshot.Node
	tabs                  []string
	podCount, podsPending int
	errs                  []string
	colTitles             [2][]string // visible columns, for "s"

	detail *podsView // pods of one node, when open

	w, h int
	help help.Model
}

// New returns a model over already started contexts.
func New(ctxs []*cluster.Context) *Model {
	return &Model{ctxs: ctxs, started: time.Now(), help: help.New(),
		sortBy: [2]string{"SAVE≈/MO", "IDLE/MO"}, desc: [2]bool{true, true}}
}

type tickMsg struct{}

func tick() tea.Cmd { return tea.Tick(time.Second, func(time.Time) tea.Msg { return tickMsg{} }) }

func (m *Model) Init() tea.Cmd { return tick() }

func (m *Model) snapshot() {
	m.snap = snapshot.Take(m.ctxs)
	m.snapAt = time.Now()
	every := refreshEvery
	if time.Since(m.started) < 15*time.Second { // the watches are still filling in
		every = 2 * time.Second
	}
	m.nextSnap = m.snapAt.Add(every)
	m.derive()
}

func poolKey(p *snapshot.Pool) string {
	if p.All {
		return "*"
	}
	return p.Ctx + "|" + p.Name
}

// derive builds the visible tables for the current tab, sort and pool from the last snapshot.
func (m *Model) derive() {
	m.pools, m.nodes, m.errs, m.podCount, m.podsPending = nil, nil, nil, 0, 0
	var pools []*snapshot.Pool
	members := map[*snapshot.Pool][]*snapshot.Node{}
	labels := []string{""}
	totalNodes, totalHourly := 0, 0.0
	for i, s := range m.snap {
		totalNodes, totalHourly = totalNodes+s.NodeCount, totalHourly+s.Hourly
		label := fmt.Sprintf("%s %dn %s", s.Name, s.NodeCount, short(s.Hourly*snapshot.HoursPerMonth))
		if s.Err != nil {
			label = s.Name + " ✗"
		}
		labels = append(labels, label)
		if m.tab != 0 && m.tab != i+1 {
			continue
		}
		m.podCount, m.podsPending = m.podCount+s.Pods, m.podsPending+s.Pending
		if s.Err != nil {
			m.errs = append(m.errs, s.Name+": "+s.Err.Error())
		}
		pools = append(pools, s.Pools...)
		for p, ns := range s.Members {
			members[p] = ns
		}
	}
	labels[0] = fmt.Sprintf("all %dn %s", totalNodes, short(totalHourly*snapshot.HoursPerMonth))
	m.tabs = labels

	all, allNodes := snapshot.AllPools(pools, members)
	sortRows(poolCols, pools, m.sortBy[0], m.desc[0])
	m.pools = append([]*snapshot.Pool{all}, pools...)
	m.restore(0)

	sel := m.pools[m.cur[0]]
	if k := poolKey(sel); k != m.shownPool {
		m.shownPool, m.cur[1], m.off[1], m.selKey[1] = k, 0, 0, ""
	}
	if sel.All {
		m.nodes = allNodes
	} else {
		m.nodes = slices.Clone(members[sel])
	}
	sortRows(nodeCols, m.nodes, m.sortBy[1], m.desc[1])
	m.restore(1)
}

func (m *Model) restore(t int) {
	if i := slices.Index(m.keysOf(t), m.selKey[t]); i >= 0 && m.selKey[t] != "" {
		m.cur[t] = i
	}
	m.cur[t] = max(0, min(m.cur[t], m.count(t)-1))
}

func (m *Model) keysOf(t int) []string {
	var out []string
	if t == 0 {
		for _, p := range m.pools {
			out = append(out, poolKey(p))
		}
	} else {
		for _, n := range m.nodes {
			out = append(out, n.Ctx+"|"+n.Name)
		}
	}
	return out
}

func (m *Model) count(t int) int {
	if t == 0 {
		return len(m.pools)
	}
	return len(m.nodes)
}

// pin records the row under each cursor so later re-sorts keep the cursor on it.
func (m *Model) pin() {
	for t := range 2 {
		m.cur[t] = max(0, min(m.cur[t], m.count(t)-1))
		m.selKey[t] = ""
		if ks := m.keysOf(t); m.cur[t] < len(ks) {
			m.selKey[t] = ks[m.cur[t]]
		}
	}
}

func (m *Model) setTab(t int) {
	n := len(m.ctxs) + 1
	m.tab = (t%n + n) % n
	m.cur, m.off, m.selKey, m.shownPool = [2]int{}, [2]int{}, [2]string{}, ""
}

func (m *Model) setSort(t int, title string) {
	if m.sortBy[t] == title {
		m.desc[t] = !m.desc[t]
		return
	}
	m.sortBy[t] = title
	numeric := false // numbers sort largest first, text alphabetically
	if t == 0 {
		numeric = poolCols[slices.IndexFunc(poolCols, func(c col[*snapshot.Pool]) bool { return c.title == title })].key != nil
	} else {
		numeric = nodeCols[slices.IndexFunc(nodeCols, func(c col[*snapshot.Node]) bool { return c.title == title })].key != nil
	}
	m.desc[t] = numeric
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
		m.help.Width = msg.Width
	case tickMsg:
		if !time.Now().Before(m.nextSnap) {
			m.snapshot()
		}
		return m, tick()
	case tea.KeyMsg:
		if m.detail != nil {
			if m.onDetailKey(msg) {
				return m, tea.Quit
			}
		} else if m.onKey(msg) {
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m *Model) onKey(msg tea.KeyMsg) (quit bool) {
	switch {
	case key.Matches(msg, keys.Quit):
		return true
	case key.Matches(msg, keys.Help):
		m.help.ShowAll = !m.help.ShowAll
	case key.Matches(msg, keys.Next):
		m.setTab(m.tab + 1)
	case key.Matches(msg, keys.Prev):
		m.setTab(m.tab - 1)
	case key.Matches(msg, keys.Tab):
		m.focus ^= 1
	case key.Matches(msg, keys.Up):
		m.cur[m.focus]--
	case key.Matches(msg, keys.Down):
		m.cur[m.focus]++
	case key.Matches(msg, keys.Enter):
		if m.focus == 0 {
			m.focus = 1
		} else {
			m.openPods()
		}
	case key.Matches(msg, keys.Sort):
		if ts := m.colTitles[m.focus]; len(ts) > 0 {
			m.setSort(m.focus, ts[(slices.Index(ts, m.sortBy[m.focus])+1)%len(ts)])
		}
	case key.Matches(msg, keys.Reverse):
		m.desc[m.focus] = !m.desc[m.focus]
	case key.Matches(msg, keys.All, keys.Esc):
		m.cur[0], m.focus = 0, 0
	case key.Matches(msg, keys.Refresh):
		for _, k := range m.ctxs {
			k.RefreshNow()
		}
		m.pin()
		m.snapshot()
		m.nextSnap = time.Now().Add(3 * time.Second) // pick up the Karpenter re-read
		return false
	}
	m.pin()
	m.derive()
	return false
}
