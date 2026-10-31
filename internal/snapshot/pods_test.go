package snapshot

import (
	"testing"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestPodsFromSumsRequests(t *testing.T) {
	req := func(cpu, mem string) v1.ResourceRequirements {
		return v1.ResourceRequirements{Requests: v1.ResourceList{v1.ResourceCPU: resource.MustParse(cpu), v1.ResourceMemory: resource.MustParse(mem)}}
	}
	pods := []v1.Pod{{
		ObjectMeta: metav1.ObjectMeta{Namespace: "ns", Name: "a"},
		Spec:       v1.PodSpec{Containers: []v1.Container{{Resources: req("250m", "1Gi")}, {Resources: req("250m", "1Gi")}}},
	}}
	rows := PodsFrom(pods)
	if len(rows) != 1 || rows[0].CPU != 0.5 || rows[0].Mem != 2<<30 || rows[0].Namespace != "ns" || rows[0].Name != "a" {
		t.Fatalf("got %+v", rows)
	}
}
