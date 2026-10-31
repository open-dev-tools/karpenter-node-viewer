package tui

import tea "github.com/charmbracelet/bubbletea"

// layout is where View last drew things, for mouse hit-testing.
type layout struct {
	tabX         []int    // start x of each tab, plus the end of the last one
	tableY, rows [2]int   // column header line and visible row count of each table
	colX         [2][]int // start x of each visible column
}

// onMouse: click a tab to switch context, a pool to show its nodes, a node to open its pods,
// a column header to sort (again to reverse); the wheel scrolls the table under the pointer.
func (m *Model) onMouse(msg tea.MouseMsg) {
	press := msg.Button == tea.MouseButtonLeft && msg.Action == tea.MouseActionPress
	if m.detail != nil {
		switch {
		case msg.Button == tea.MouseButtonWheelUp:
			m.detail.off -= 3
		case msg.Button == tea.MouseButtonWheelDown:
			m.detail.off += 3
		case press:
			m.detail = nil
		}
		return
	}
	table, row, header := -1, -1, false
	for t := range 2 {
		switch y := msg.Y - m.at.tableY[t]; {
		case y == -1 || y == 0: // title bar or column header
			table, header = t, y == 0
		case y > 0 && y <= m.at.rows[t]:
			table, row = t, m.off[t]+y-1
		}
	}
	switch {
	case msg.Button == tea.MouseButtonWheelUp && table >= 0:
		m.focus, m.cur[table] = table, m.cur[table]-3
	case msg.Button == tea.MouseButtonWheelDown && table >= 0:
		m.focus, m.cur[table] = table, m.cur[table]+3
	case !press:
		return
	case msg.Y == 0:
		for i := 0; i+1 < len(m.at.tabX); i++ {
			if msg.X >= m.at.tabX[i] && msg.X < m.at.tabX[i+1] {
				m.setTab(i)
			}
		}
	case header:
		m.focus = table
		c := 0
		for i, x := range m.at.colX[table] {
			if msg.X >= x {
				c = i
			}
		}
		if c < len(m.colTitles[table]) {
			m.setSort(table, m.colTitles[table][c])
		}
	case table >= 0 && row < 0:
		m.focus = table
	case table >= 0 && row < m.count(table):
		m.focus, m.cur[table] = table, row
		if table == 1 {
			m.pin()
			m.openPods()
		}
	}
	m.pin()
	m.derive()
}
