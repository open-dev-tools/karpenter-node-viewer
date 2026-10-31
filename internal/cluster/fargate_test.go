package cluster

import (
	"math"
	"testing"

	"github.com/awslabs/eks-node-viewer/pkg/model"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type panicProvider struct{}

func (panicProvider) NodePrice(*model.Node) (float64, bool) {
	panic("must not be called for Fargate nodes")
}
func (panicProvider) OnUpdate(func()) {}

// FargateNode builds a Fargate node whose only pod carries the given CapacityProvisioned annotation.
func FargateNode(annotation string) (*model.Cluster, *model.Node) {
	c := model.NewCluster()
	n := c.AddNode(model.NewNode(&v1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: "fargate-ip-10-0-0-2", Labels: map[string]string{"eks.amazonaws.com/compute-type": "fargate"}},
		Spec:       v1.NodeSpec{ProviderID: "aws:///us-east-1a/fargate-ip-10-0-0-2"},
	}))
	c.AddPod(model.NewPod(&v1.Pod{
		ObjectMeta: metav1.ObjectMeta{Namespace: "team", Name: "p", Annotations: map[string]string{"CapacityProvisioned": annotation}},
		Spec:       v1.PodSpec{NodeName: "fargate-ip-10-0-0-2"},
		Status:     v1.PodStatus{Phase: v1.PodRunning},
	}))
	return c, n
}

func TestNoFargateSkipsPricing(t *testing.T) {
	for _, a := range []string{"bogus", "-8vCPU -16GB", "0.25vCPU 0.5GB"} {
		_, n := FargateNode(a)
		if p, ok := (noFargate{panicProvider{}}).NodePrice(n); ok || !math.IsNaN(p) {
			t.Errorf("annotation %q: got (%v, %v), want (NaN, false)", a, p, ok)
		}
	}
}
