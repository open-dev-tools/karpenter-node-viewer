package tui

import (
	"fmt"
	"math"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// cell is one table cell. s is the plain text used for width and for sorting text columns.
type cell struct {
	s     string
	st    lipgloss.Style
	right bool
	draw  func() string // when set, draws the cell instead of st (gauges mix colors)
}

// col is a table column over rows of type T.
type col[T any] struct {
	title string
	key   func(T) float64 // numeric sort key; nil sorts by cell text
	cell  func(T) cell
	flex  bool // takes the remaining width
}

const (
	barW   = 10
	gutter = 2 // room for the cursor marker in front of each row
)

var (
	cGood, cOK, cBad = lipgloss.Color("#04B575"), lipgloss.Color("#FFFF00"), lipgloss.Color("#FF0000")
	cAccent          = lipgloss.Color("#7D56F4")
	bold             = lipgloss.NewStyle().Bold(true)
	dim              = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	warn             = lipgloss.NewStyle().Foreground(cOK)
	errS             = lipgloss.NewStyle().Foreground(cBad)
	marker           = lipgloss.NewStyle().Foreground(cAccent).Bold(true)
	barFill          = lipgloss.NewStyle().Foreground(lipgloss.Color("250"))
	barRest          = lipgloss.NewStyle().Foreground(lipgloss.Color("238"))
	accentBar        = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFFFFF")).Background(cAccent)
	grayBar          = lipgloss.NewStyle().Foreground(lipgloss.Color("250")).Background(lipgloss.Color("237"))
)

// bar is a thin gray gauge with a colored percentage. Low is red: an under-requested node is the
// waste this tool looks for.
func bar(p float64) cell {
	n := int(math.Round(math.Min(1, math.Max(0, p)) * barW))
	c := cBad
	if p >= .7 {
		c = cGood
	} else if p >= .4 {
		c = cOK
	}
	pct := fmt.Sprintf(" %3.0f%%", p*100)
	return cell{s: strings.Repeat("━", barW) + pct, draw: func() string {
		return barFill.Render(strings.Repeat("━", n)) + barRest.Render(strings.Repeat("─", barW-n)) + lipgloss.NewStyle().Foreground(c).Render(pct)
	}}
}

func money(x float64) string {
	if math.IsNaN(x) {
		return "?"
	}
	s := fmt.Sprintf("%.0f", math.Abs(x))
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	if x < 0 {
		return "-$" + s
	}
	return "$" + s
}

func short(x float64) string {
	if x >= 1000 {
		return fmt.Sprintf("$%.1fk", x/1000)
	}
	return money(x)
}

func num(s string) cell { return cell{s: s, right: true} }
func txt(s string) cell { return cell{s: s} }

func age(t time.Time) string {
	d := time.Since(t)
	switch {
	case d >= 24*time.Hour:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	case d >= time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
	return fmt.Sprintf("%dm", int(d.Minutes()))
}

// pick drops the columns whose titles are in hide (columns implied by the current selection).
func pick[T any](cols []col[T], hide ...string) []col[T] {
	return slices.DeleteFunc(slices.Clone(cols), func(c col[T]) bool { return slices.Contains(hide, c.title) })
}

func titles[T any](cols []col[T]) []string {
	out := make([]string, len(cols))
	for i, c := range cols {
		out[i] = c.title
	}
	return out
}

func sortRows[T any](cols []col[T], rows []T, by string, desc bool) {
	i := slices.IndexFunc(cols, func(c col[T]) bool { return c.title == by })
	if i < 0 {
		return
	}
	cl := cols[i]
	less := func(a, b T) bool {
		if cl.key != nil {
			return nan2inf(cl.key(a)) < nan2inf(cl.key(b))
		}
		return cl.cell(a).s < cl.cell(b).s
	}
	sort.SliceStable(rows, func(i, j int) bool {
		if desc {
			return less(rows[j], rows[i])
		}
		return less(rows[i], rows[j])
	})
}

func nan2inf(x float64) float64 {
	if math.IsNaN(x) {
		return math.Inf(-1)
	}
	return x
}

// renderTable returns the header line plus height row lines, and the x where each column starts.
func renderTable[T any](cols []col[T], rows []T, sortBy string, desc bool, cur, off, height, width int, focused bool) ([]string, []int) {
	all := make([][]cell, len(rows))
	widths := make([]int, len(cols))
	for i, c := range cols {
		widths[i] = len([]rune(c.title)) + 2 // room for the sort arrow
	}
	for r, row := range rows { // size over every row so columns do not jump while scrolling
		all[r] = make([]cell, len(cols))
		for i, c := range cols {
			all[r][i] = c.cell(row)
			if !c.flex {
				widths[i] = max(widths[i], len([]rune(all[r][i].s)))
			}
		}
	}
	used, flex := gutter, -1
	for i, c := range cols {
		if c.flex {
			flex = i
		} else {
			used += widths[i] + 1
		}
	}
	if flex >= 0 {
		widths[flex] = max(0, width-used)
	}
	xs := make([]int, len(cols))
	hb := strings.Repeat(" ", gutter)
	x := gutter
	for i, c := range cols {
		xs[i] = x
		t := c.title
		if t == sortBy {
			t += map[bool]string{true: " ▼", false: " ▲"}[desc]
		}
		hb += bold.Render(pad(trunc(t, widths[i]), widths[i], false)) + " "
		x += widths[i] + 1
	}
	lines := []string{hb}
	for r := off; r < off+height; r++ {
		if r >= len(rows) {
			lines = append(lines, "")
			continue
		}
		var b strings.Builder
		switch {
		case r == cur && focused:
			b.WriteString(marker.Render("▶ "))
		case r == cur:
			b.WriteString(dim.Render("▷ "))
		default:
			b.WriteString("  ")
		}
		for i := range cols {
			if widths[i] == 0 {
				continue
			}
			c := all[r][i]
			if c.draw != nil {
				b.WriteString(c.draw() + strings.Repeat(" ", widths[i]-len([]rune(c.s))+1))
				continue
			}
			st := c.st
			if r == cur && focused {
				st = st.Bold(true)
			}
			b.WriteString(st.Render(pad(trunc(c.s, widths[i]), widths[i], c.right)) + " ")
		}
		lines = append(lines, b.String())
	}
	return lines, xs
}

func pad(s string, w int, right bool) string {
	gap := strings.Repeat(" ", max(0, w-len([]rune(s))))
	if right {
		return gap + s
	}
	return s + gap
}

func trunc(s string, w int) string {
	if r := []rune(s); len(r) > w {
		if w <= 0 {
			return ""
		}
		return string(r[:w-1]) + "…"
	}
	return s
}

// section is the colored title bar above a table. The focused table's bar uses the accent color.
func section(w int, on bool, left, right string) string {
	st := grayBar
	if on {
		st = accentBar
	}
	l, r := len([]rune(left)), len([]rune(right))
	if l+r+1 > w {
		right, r = "", 0
	}
	return st.Render(trunc(left+strings.Repeat(" ", max(1, w-l-r))+right, w))
}
