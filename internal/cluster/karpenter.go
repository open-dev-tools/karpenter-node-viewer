package cluster

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PoolSpec is the disruption block of a Karpenter NodePool.
type PoolSpec struct {
	Policy, After, Budget string // consolidationPolicy, consolidateAfter, budgets[].nodes
}

const pollEvery = 30 * time.Second

// Karpenter returns the latest NodePool settings and, per node name, the latest message of
// Karpenter's Unconsolidatable or DisruptionBlocked events. err is the connection or last poll error.
func (k *Context) Karpenter() (specs map[string]PoolSpec, why map[string]string, err error) {
	k.mu.Lock()
	defer k.mu.Unlock()
	return k.specs, k.why, k.err
}

// RefreshNow asks the poller to re-read Karpenter state without waiting for the next tick.
func (k *Context) RefreshNow() {
	select {
	case k.kick <- struct{}{}:
	default:
	}
}

func (k *Context) poll(ctx context.Context) {
	for {
		specs, why, err := k.fetch(ctx)
		k.mu.Lock()
		if err == nil {
			k.specs, k.why = specs, why
		}
		k.err = err
		k.mu.Unlock()
		select {
		case <-ctx.Done():
			return
		case <-k.kick:
		case <-time.After(pollEvery):
		}
	}
}

func (k *Context) fetch(ctx context.Context) (map[string]PoolSpec, map[string]string, error) {
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	specs := map[string]PoolSpec{}
	raw, err := k.karp.Get().Resource("nodepools").DoRaw(ctx)
	if err != nil && !apierrors.IsNotFound(err) { // no Karpenter: only managed node groups show up
		return nil, nil, fmt.Errorf("nodepools: %w", err)
	}
	if err == nil {
		var list struct {
			Items []struct {
				Metadata struct{ Name string } `json:"metadata"`
				Spec     struct {
					Disruption struct {
						ConsolidationPolicy string `json:"consolidationPolicy"`
						ConsolidateAfter    string `json:"consolidateAfter"`
						Budgets             []struct {
							Nodes string `json:"nodes"`
						} `json:"budgets"`
					} `json:"disruption"`
				} `json:"spec"`
			} `json:"items"`
		}
		if err := json.Unmarshal(raw, &list); err != nil {
			return nil, nil, fmt.Errorf("nodepools: %w", err)
		}
		for _, np := range list.Items {
			d := np.Spec.Disruption
			var b []string
			for _, x := range d.Budgets {
				b = append(b, x.Nodes)
			}
			specs[np.Metadata.Name] = PoolSpec{or(d.ConsolidationPolicy), or(d.ConsolidateAfter), or(strings.Join(b, ","))}
		}
	}

	// Karpenter records each event on both the Node and the NodeClaim, in the default namespace.
	// The Node copy is enough.
	evs, err := k.kube.CoreV1().Events("default").List(ctx, metav1.ListOptions{FieldSelector: "involvedObject.kind=Node"})
	if err != nil {
		return nil, nil, fmt.Errorf("events: %w", err)
	}
	why, seen := map[string]string{}, map[string]time.Time{}
	for _, e := range evs.Items {
		if e.Reason != "Unconsolidatable" && e.Reason != "DisruptionBlocked" {
			continue
		}
		if t := eventTime(e); t.After(seen[e.InvolvedObject.Name]) {
			seen[e.InvolvedObject.Name], why[e.InvolvedObject.Name] = t, e.Message
		}
	}
	return specs, why, nil
}

func eventTime(e v1.Event) time.Time {
	switch {
	case e.Series != nil:
		return e.Series.LastObservedTime.Time
	case !e.LastTimestamp.IsZero():
		return e.LastTimestamp.Time
	case !e.EventTime.IsZero():
		return e.EventTime.Time
	}
	return e.CreationTimestamp.Time
}

func or(s string) string {
	if s == "" {
		return "-"
	}
	return s
}
