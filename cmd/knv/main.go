// Command knv shows Karpenter NodePool bin-packing, cost and consolidation blockers
// across kube contexts in one terminal screen.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/go-logr/logr"
	"k8s.io/klog/v2"

	"github.com/open-dev-tools/karpenter-node-viewer/internal/cluster"
	"github.com/open-dev-tools/karpenter-node-viewer/internal/tui"
)

func main() {
	kc := filepath.Join(os.Getenv("HOME"), ".kube", "config")
	if env := os.Getenv("KUBECONFIG"); env != "" {
		kc = filepath.SplitList(env)[0]
	}
	// controller-runtime registers -kubeconfig on the global FlagSet in init, so use our own
	fs := flag.NewFlagSet("knv", flag.ExitOnError)
	kubeconfig := fs.String("kubeconfig", kc, "kubeconfig path")
	contexts := fs.String("context", "", "comma-separated kube contexts (default: every context whose kubeconfig user's exec command is 'aws')")
	_ = fs.Parse(os.Args[1:])

	var only []string
	for _, c := range strings.Split(*contexts, ",") {
		if c = strings.TrimSpace(c); c != "" {
			only = append(only, c)
		}
	}
	targets, err := cluster.Targets(*kubeconfig, only)
	if err != nil || len(targets) == 0 {
		fmt.Fprintln(os.Stderr, "knv: no kube contexts:", err)
		os.Exit(1)
	}

	// client-go and pricing logs would draw over the TUI; connection errors show in the tabs instead
	klog.SetLogger(logr.Discard())
	log.SetOutput(io.Discard)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var ctxs []*cluster.Context
	for _, t := range targets {
		ctxs = append(ctxs, cluster.Start(ctx, *kubeconfig, t))
	}
	if _, err := tea.NewProgram(tui.New(ctxs), tea.WithAltScreen(), tea.WithMouseCellMotion()).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "knv:", err)
		os.Exit(1)
	}
}
