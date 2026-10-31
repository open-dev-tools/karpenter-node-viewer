package snapshot

import (
	"testing"

	"github.com/awslabs/eks-node-viewer/pkg/model"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-dev-tools/karpenter-node-viewer/internal/cluster"
)

// A malformed CapacityProvisioned annotation on a Fargate node's only pod must not crash the snapshot.
func TestTakeSurvivesMalformedFargateAnnotation(t *testing.T) {
	c := model.NewCluster()
	c.AddNode(model.NewNode(&v1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: "fargate-ip-10-0-0-2", Labels: map[string]string{"eks.amazonaws.com/compute-type": "fargate"}},
		Spec:       v1.NodeSpec{ProviderID: "aws:///us-east-1a/fargate-ip-10-0-0-2"},
	}))
	c.AddPod(model.NewPod(&v1.Pod{
		ObjectMeta: metav1.ObjectMeta{Namespace: "team", Name: "p", Annotations: map[string]string{"CapacityProvisioned": "bogus"}},
		Spec:       v1.PodSpec{NodeName: "fargate-ip-10-0-0-2"},
		Status:     v1.PodStatus{Phase: v1.PodRunning},
	}))
	snap := Take([]*cluster.Context{{Name: "ctx", Cluster: c}})
	if len(snap) != 1 || len(snap[0].Pools) != 1 {
		t.Fatalf("unexpected snapshot %+v", snap)
	}
	n := snap[0].Members[snap[0].Pools[0]][0]
	if n.InstanceType != "Fargate" {
		t.Errorf("InstanceType = %q, want Fargate", n.InstanceType)
	}
}
