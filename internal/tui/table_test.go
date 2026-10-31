package tui

import (
	"math"
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
