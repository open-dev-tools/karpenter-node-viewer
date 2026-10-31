// Package cluster connects to EKS clusters and keeps a live view of their nodes and pods.
//
// Watching, request accounting and pricing come from eks-node-viewer
// (github.com/awslabs/eks-node-viewer): one eks-node-viewer controller runs per kube context.
package cluster

import (
	"context"
	"slices"
	"sort"
	"sync"

	awscfg "github.com/aws/aws-sdk-go-v2/config"
	"github.com/awslabs/eks-node-viewer/pkg/aws"
	"github.com/awslabs/eks-node-viewer/pkg/client"
	"github.com/awslabs/eks-node-viewer/pkg/model"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Target is a kube context plus the AWS profile and region its kubeconfig user signs in with.
type Target struct {
	Name, Profile, Region string
}

// Targets reads kubeconfig and returns the contexts to watch, sorted by name.
// With no names given it picks every context whose user runs `aws eks get-token`;
// the AWS profile and region come from that exec block (AWS_PROFILE env, --region arg).
func Targets(kubeconfig string, only []string) ([]Target, error) {
	rules := clientcmd.NewDefaultClientConfigLoadingRules()
	rules.ExplicitPath = kubeconfig
	cfg, err := rules.Load()
	if err != nil {
		return nil, err
	}
	var out []Target
	for name, c := range cfg.Contexts {
		t := Target{Name: name}
		ai := cfg.AuthInfos[c.AuthInfo]
		eks := ai != nil && ai.Exec != nil && ai.Exec.Command == "aws"
		if eks {
			for _, e := range ai.Exec.Env {
				if e.Name == "AWS_PROFILE" {
					t.Profile = e.Value
				}
			}
			for i, a := range ai.Exec.Args {
				if a == "--region" && i+1 < len(ai.Exec.Args) {
					t.Region = ai.Exec.Args[i+1]
				}
			}
		}
		if (len(only) == 0 && eks) || slices.Contains(only, name) {
			out = append(out, t)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// Context is one watched kube context.
type Context struct {
	Name    string
	Cluster *model.Cluster // live nodes and pods, kept current by eks-node-viewer

	kube *kubernetes.Clientset
	karp *rest.RESTClient // karpenter.sh/v1
	kick chan struct{}

	mu    sync.Mutex
	specs map[string]PoolSpec // NodePool name -> disruption settings
	why   map[string]string   // node name -> latest consolidation blocker
	err   error
}

// Start connects to t and starts watching nodes, pods and NodeClaims.
// A connection error is kept on the returned Context instead of aborting,
// so one broken context does not hide the others.
func Start(ctx context.Context, kubeconfig string, t Target) *Context {
	k := &Context{Name: t.Name, Cluster: model.NewCluster(), kick: make(chan struct{}, 1)}
	var err error
	if k.kube, err = client.NewKubernetes(kubeconfig, t.Name); err == nil {
		k.karp, err = client.NewNodeClaims(kubeconfig, t.Name)
	}
	if err != nil {
		k.err = err
		return k
	}

	// On-demand prices start from the table built into eks-node-viewer and are refreshed from the
	// AWS Pricing API; spot prices come from EC2 spot price history for the context's region.
	opts := []func(*awscfg.LoadOptions) error{awscfg.WithRegion(t.Region)}
	if t.Profile != "" {
		opts = append(opts, awscfg.WithSharedConfigProfile(t.Profile))
	}
	pricing := aws.NewStaticPricingProvider()
	if cfg, err := awscfg.LoadDefaultConfig(ctx, opts...); err == nil {
		pricing = aws.NewPricingProvider(ctx, cfg)
	}

	// The controller writes into a UIModel; knv only uses it as a data holder and draws its own screen.
	style, _ := model.ParseStyle("#04b575,#ffff00,#ff0000") // constant input, cannot fail
	holder := model.NewUIModel(nil, "creation", style)
	k.Cluster = holder.Cluster()
	client.NewController(k.kube, k.karp, holder, labels.Everything(), pricing).Start(ctx)
	go k.poll(ctx)
	return k
}
