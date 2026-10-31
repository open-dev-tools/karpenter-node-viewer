package cluster

import (
	"testing"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func event(node, reason, msg, component string, created, last time.Time) v1.Event {
	return v1.Event{
		ObjectMeta:     metav1.ObjectMeta{CreationTimestamp: metav1.NewTime(created)},
		InvolvedObject: v1.ObjectReference{Kind: "Node", Name: node},
		Reason:         reason, Message: msg,
		Source:        v1.EventSource{Component: component},
		LastTimestamp: metav1.NewTime(last),
	}
}

// A blocker that Karpenter reported must survive events from other reporters, empty messages
// and author-supplied future timestamps.
func TestLatestBlockersTrustsOnlyKarpenter(t *testing.T) {
	now := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	future := time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC)
	genuine := `Pdb prevents pod evictions (PodDisruptionBudget=[shop/checkout])`
	items := []v1.Event{
		event("node-a", "Unconsolidatable", genuine, "karpenter", now.Add(-time.Minute), now.Add(-time.Minute)),
		event("node-a", "Unconsolidatable", "", "karpenter", now.Add(-30*time.Second), future),            // empty
		event("node-a", "DisruptionBlocked", "forged", "not-karpenter", now.Add(-10*time.Second), future), // other reporter
		event("node-b", "DisruptionBlocked", "pinned", "karpenter", now.Add(-time.Hour), future),          // future timestamp
		event("node-b", "DisruptionBlocked", "newer", "karpenter", now.Add(-time.Minute), now.Add(-time.Minute)),
		event("node-c", "Started", "ignored", "karpenter", now, now),
	}
	got := latestBlockers(items, now)
	if got["node-a"] != genuine {
		t.Errorf("node-a = %q, want the Karpenter PDB message", got["node-a"])
	}
	if got["node-b"] != "newer" {
		t.Errorf("node-b = %q, a future timestamp must not pin an older event", got["node-b"])
	}
	if _, ok := got["node-c"]; ok {
		t.Errorf("node-c has a blocker from a non-blocker reason")
	}
}
