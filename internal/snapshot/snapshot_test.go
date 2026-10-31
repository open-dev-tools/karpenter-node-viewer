package snapshot

import (
	"math"
	"testing"
)

func TestFitAndSave(t *testing.T) {
	// 4 nodes of 4 CPU / 16Gi, 1 CPU / 2Gi requested on each: 4 CPU over an average of 4 -> 1 node
	if got := fit(4, 16, 4, 64<<30, 8<<30); got != 1 {
		t.Fatalf("cpu-bound fit = %d", got)
	}
	// memory-bound: three 16Gi nodes, 40Gi requested -> 3
	if got := fit(3, 12, 1, 48<<30, 40<<30); got != 3 {
		t.Fatalf("mem-bound fit = %d", got)
	}
	// an exact multiple is not rounded up
	if got := fit(4, 16, 8, 64, 8); got != 2 {
		t.Fatalf("exact fit = %d", got)
	}
	if got := fit(2, 8, -4, 32, -1); got != 0 {
		t.Fatalf("negative requests fit = %d", got)
	}
	if got := fit(2, 8, 0, 32, 0); got != 0 {
		t.Fatalf("empty fit = %d", got)
	}
	if s := save(4, 1, 4, 0.8); math.Abs(s-3*0.2*HoursPerMonth) > 1e-9 {
		t.Fatalf("save = %f", s)
	}
	if s := save(4, 1, 0, 0); !math.IsNaN(s) {
		t.Fatalf("unpriced save = %f", s)
	}
}

func TestShortWhy(t *testing.T) {
	for msg, want := range map[string][2]string{
		`Pod has "karpenter.sh/do-not-disrupt" annotation (Pod=batch/report-7f9c)`: {"do-not-disrupt pod batch/report-7f9c", "do-not-disrupt pod"},
		`Pdb prevents pod evictions (PodDisruptionBudget=[shop/checkout])`:         {"PDB shop/checkout", "PDB blocks eviction"},
		`NodePool "general" has non-empty consolidation disabled`:                  {"non-empty consolidation disabled", "non-empty consolidation disabled"},
		`Can't replace with a cheaper node`:                                        {"Can't replace with a cheaper node", "Can't replace with a cheaper node"},
		``:                                                                         {"", ""},
	} {
		if s, k := shortWhy(msg); s != want[0] || k != want[1] {
			t.Errorf("shortWhy(%q) = %q, %q", msg, s, k)
		}
	}
}

func TestNodeGroupSuffix(t *testing.T) {
	if got := ngSuffix.ReplaceAllString("eks-ng-system-20260319053047507700000018", ""); got != "eks-ng-system" {
		t.Fatalf("got %q", got)
	}
}
