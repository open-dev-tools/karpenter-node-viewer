package tui

import (
	"math"
	"strings"
	"testing"
)

func TestMoney(t *testing.T) {
	for in, want := range map[float64]string{6117.4: "$6,117", 12: "$12", 1234567: "$1,234,567", -1500: "-$1,500", math.NaN(): "?"} {
		if got := money(in); got != want {
			t.Errorf("money(%v) = %q, want %q", in, got, want)
		}
	}
}

func TestTruncPad(t *testing.T) {
	if got := trunc("general-on-demand", 8); got != "general…" {
		t.Errorf("trunc = %q", got)
	}
	if got := trunc("abc", 0); got != "" {
		t.Errorf("trunc to zero = %q", got)
	}
	if got := pad("7", 3, true); got != "  7" {
		t.Errorf("pad right = %q", got)
	}
}

func TestSortRowsKeepsNaNLast(t *testing.T) {
	type row struct{ v float64 }
	cols := []col[row]{{title: "V", key: func(r row) float64 { return r.v }, cell: func(row) cell { return cell{} }}}
	rows := []row{{1}, {math.NaN()}, {3}}
	sortRows(cols, rows, "V", true)
	if rows[0].v != 3 || rows[1].v != 1 || !math.IsNaN(rows[2].v) {
		t.Errorf("got %v", rows)
	}
}

// Cluster-supplied text must not reach the terminal as control sequences (OSC 52 clipboard, CSI, newlines, bidi overrides).
func TestRenderNeutralizesControlRunes(t *testing.T) {
	type row struct{ s string }
	payload := "a\x1b]52;c;ZHVtbXk=\x07b\nc\u009bd\u202ee"
	cols := []col[row]{{title: "WHY", cell: func(r row) cell { return txt(r.s) }}}
	lines, _ := renderTable(cols, []row{{payload}}, "", false, -1, 0, 1, 80, false)
	out := strings.Join(lines, "\n") + section(80, true, payload, payload)
	for _, bad := range []string{"\x1b]52", "\x07", "b\nc", "\u009b", "\u202e"} {
		if strings.Contains(out, bad) {
			t.Errorf("rendered output still contains %q", bad)
		}
	}
	if !strings.Contains(out, "a?]52;c;ZHVtbXk=?b?c?d?e") {
		t.Errorf("expected neutralized text, got %q", out)
	}
}
