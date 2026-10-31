// Package snapshot freezes the live cluster state into NodePool and node rows.
//
// Taking a snapshot is the only place knv reads the clusters; everything the screen does between
// snapshots (cursor, sort, filter) works on these rows, so rows do not move under the cursor.
package snapshot

import (
	"fmt"
	"math"
	"regexp"
	"strings"
	"time"

	"github.com/awslabs/eks-node-viewer/pkg/model"
	v1 "k8s.io/api/core/v1"

	"github.com/open-dev-tools/karpenter-node-viewer/internal/cluster"
)

// HoursPerMonth converts hourly prices to monthly, the same way eks-node-viewer does.
const HoursPerMonth = 730

// Node is one node at snapshot time. CPU and memory are requests over allocatable, not usage.
type Node struct {
	Ctx, Name, Pool, InstanceType, Capacity, State string
	Why                                            string // latest consolidation blocker, shortened
	WhyKind                                        string // Why without the pod or PDB name, for counting per pool

	CPUAlloc, CPUReq, MemAlloc, MemReq, CPUPct, MemPct float64
	Price, Idle                                        float64 // $/h and idle $/month; NaN when the price is unknown
	Pods                                               int
	Created                                            time.Time

	live *model.Node
}

// Pool is one NodePool (or managed node group) in one context.
type Pool struct {
	Ctx, Name string
	Spec      cluster.PoolSpec

	Nodes, Fit                    int
	CPUPct, MemPct, Monthly, Save float64
	Blocker                       string // most common blocker, "n/total reason"
	All                           bool   // the "(all pools)" total row
}

// Context is one kube context at snapshot time.
type Context struct {
	Name          string
	Pools         []*Pool
	Members       map[*Pool][]*Node
	Pods, Pending int
	NodeCount     int
	Hourly        float64
	Err           error
}

// Take reads every context once.
func Take(ctxs []*cluster.Context) []Context {
	out := make([]Context, 0, len(ctxs))
	for _, k := range ctxs {
		specs, why, err := k.Karpenter()
		st := k.Cluster.Stats()
		c := Context{Name: k.Name, Members: map[*Pool][]*Node{}, Pods: st.TotalPods,
			Pending: st.PodsByPhase[v1.PodPending], NodeCount: st.NumNodes, Hourly: st.TotalPrice, Err: err}
		idx := map[string]*Pool{}
		k.Cluster.ForEachNode(func(n *model.Node) {
			r := newNode(k.Name, n, why)
			p := idx[r.Pool]
			if p == nil {
				spec, ok := specs[r.Pool]
				if !ok {
					spec = cluster.PoolSpec{Policy: "-", After: "-", Budget: "-"}
					if strings.HasPrefix(r.Pool, "mng/") {
						spec.Policy = "MNG"
					}
				}
				p = &Pool{Ctx: k.Name, Name: r.Pool, Spec: spec}
				idx[r.Pool] = p
				c.Pools = append(c.Pools, p)
			}
			c.Members[p] = append(c.Members[p], r)
		})
		for _, p := range c.Pools {
			summarize(p, c.Members[p])
		}
		out = append(out, c)
	}
	return out
}

// AllPools returns the "(all pools)" total row and every node of the given pools.
func AllPools(pools []*Pool, members map[*Pool][]*Node) (*Pool, []*Node) {
	all := &Pool{Name: "(all pools)", All: true}
	var nodes []*Node
	var a agg
	for _, p := range pools {
		for _, n := range members[p] {
			a.add(n)
		}
		all.Fit += p.Fit
		if !math.IsNaN(p.Save) {
			all.Save += p.Save
		}
		nodes = append(nodes, members[p]...)
	}
	all.Nodes = len(nodes)
	all.CPUPct, all.MemPct, all.Monthly = ratio(a.cpuReq, a.cpuAlloc), ratio(a.memReq, a.memAlloc), a.hourly*HoursPerMonth
	return all, nodes
}

func summarize(p *Pool, nodes []*Node) {
	var a agg
	kinds := map[string]int{}
	for _, n := range nodes {
		a.add(n)
		if n.WhyKind != "" {
			kinds[n.WhyKind]++
		}
	}
	p.Nodes = len(nodes)
	p.CPUPct, p.MemPct, p.Monthly = ratio(a.cpuReq, a.cpuAlloc), ratio(a.memReq, a.memAlloc), a.hourly*HoursPerMonth
	p.Fit = fit(p.Nodes, a.cpuAlloc, a.cpuReq, a.memAlloc, a.memReq)
	p.Save = save(p.Nodes, p.Fit, a.priced, a.hourly)
	best := 0
	for k, c := range kinds {
		if c > best || (c == best && k < p.Blocker) {
			p.Blocker, best = k, c
		}
	}
	if best > 0 {
		p.Blocker = fmt.Sprintf("%d/%d %s", best, p.Nodes, p.Blocker)
	}
}

type agg struct {
	cpuAlloc, cpuReq, memAlloc, memReq, hourly float64
	priced                                     int
}

func (a *agg) add(n *Node) {
	a.cpuAlloc, a.cpuReq, a.memAlloc, a.memReq = a.cpuAlloc+n.CPUAlloc, a.cpuReq+n.CPUReq, a.memAlloc+n.MemAlloc, a.memReq+n.MemReq
	if !math.IsNaN(n.Price) {
		a.hourly, a.priced = a.hourly+n.Price, a.priced+1
	}
}

// fit is how many average-sized nodes the pool's requests would need if packed perfectly (a lower bound).
// It ignores affinity, topology spread, PDBs and do-not-disrupt, so pools that pin one pod per node
// show savings they cannot reach. DaemonSet requests are not subtracted, which errs on the safe side.
func fit(n int, cpuAlloc, cpuReq, memAlloc, memReq float64) int {
	if n == 0 || cpuAlloc == 0 || memAlloc == 0 {
		return n
	}
	need := math.Max(cpuReq/(cpuAlloc/float64(n)), memReq/(memAlloc/float64(n)))
	return min(n, int(math.Ceil(need-1e-9)))
}

// save is the monthly saving of shrinking the pool to fit nodes, at the pool's average node price.
func save(n, fit, priced int, hourly float64) float64 {
	if priced == 0 {
		return math.NaN()
	}
	return float64(n-fit) * hourly / float64(priced) * HoursPerMonth
}

var (
	// the 26-character suffix terraform's name_prefix appends to managed node group names
	ngSuffix   = regexp.MustCompile(`-[0-9a-z]{20,}$`)
	poolPrefix = regexp.MustCompile(`^NodePool "[^"]+" has `) // repeats the NODEPOOL column
	dndRe      = regexp.MustCompile(`^Pod has "karpenter\.sh/do-not-disrupt" annotation \(Pod=(.+)\)$`)
	pdbRe      = regexp.MustCompile(`^Pdb prevents pod evictions \(PodDisruptionBudget=\[?(.+?)\]?\)$`)
)

// shortWhy shortens a Karpenter blocker message. The pod or PDB name goes last so the reason
// survives truncation; kind drops the name so a pool counts the same reason once per node.
func shortWhy(msg string) (short, kind string) {
	msg = poolPrefix.ReplaceAllString(msg, "")
	if m := dndRe.FindStringSubmatch(msg); m != nil {
		return "do-not-disrupt pod " + m[1], "do-not-disrupt pod"
	}
	if m := pdbRe.FindStringSubmatch(msg); m != nil {
		return "PDB " + m[1], "PDB blocks eviction"
	}
	return msg, msg
}

func newNode(ctx string, n *model.Node, why map[string]string) *Node {
	lb := n.Labels()
	pool := lb["karpenter.sh/nodepool"]
	if ng := lb["eks.amazonaws.com/nodegroup"]; pool == "" && ng != "" {
		pool = "mng/" + ngSuffix.ReplaceAllString(ng, "")
	}
	r := &Node{Ctx: ctx, Name: n.Name(), Pool: or(pool), InstanceType: or(string(n.InstanceType())), Capacity: "?",
		Pods: n.NumPods(), Created: n.Created(), Price: math.NaN(), Idle: math.NaN(), live: n}
	r.Why, r.WhyKind = shortWhy(why[n.Name()])
	switch {
	case n.IsSpot():
		r.Capacity = "spot"
	case n.IsOnDemand():
		r.Capacity = "on-demand"
	}
	switch {
	case n.Deleting():
		r.State = "Deleting"
	case n.Cordoned():
		r.State = "Cordoned"
	case !n.Ready():
		r.State = "NotReady"
	}
	alloc, used := n.Allocatable(), n.Used()
	r.CPUAlloc, r.CPUReq = res(alloc, v1.ResourceCPU), res(used, v1.ResourceCPU)
	r.MemAlloc, r.MemReq = res(alloc, v1.ResourceMemory), res(used, v1.ResourceMemory)
	r.CPUPct, r.MemPct = ratio(r.CPUReq, r.CPUAlloc), ratio(r.MemReq, r.MemAlloc)
	if n.HasPrice() {
		r.Price = n.Price
		r.Idle = r.Price * (1 - math.Min(1, math.Max(r.CPUPct, r.MemPct))) * HoursPerMonth
	}
	return r
}

// Pod is one pod's requests on a node.
type Pod struct {
	Namespace, Name string
	CPU, Mem        float64
}

// PodList returns the pods on the node now (read live, not from the snapshot).
func (n *Node) PodList() []Pod {
	var out []Pod
	for _, p := range n.live.Pods() {
		req := p.Requested()
		out = append(out, Pod{p.Namespace(), p.Name(), res(req, v1.ResourceCPU), res(req, v1.ResourceMemory)})
	}
	return out
}

func res(l v1.ResourceList, name v1.ResourceName) float64 {
	q := l[name]
	return q.AsApproximateFloat64()
}

func ratio(a, b float64) float64 {
	if b == 0 {
		return 0
	}
	return a / b
}

func or(s string) string {
	if s == "" {
		return "-"
	}
	return s
}
