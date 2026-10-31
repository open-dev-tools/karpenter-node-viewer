# karpenter-node-viewer (`knv`)

Karpenter NodePools across several EKS contexts on one terminal screen — how under-filled they are, how many nodes they actually need, and what is blocking consolidation.

![Go](https://img.shields.io/badge/Go-1.26+-00ADD8)
![License](https://img.shields.io/badge/License-MIT-yellow)

[한국어](README.md) | **English**

## Why

[eks-node-viewer](https://github.com/awslabs/eks-node-viewer) shows nodes well, but one context at a time, node by node, and not why Karpenter is not removing them.
knv reuses eks-node-viewer's data layer (watches, request accounting, pricing) and adds a per-NodePool summary and the consolidation blockers on top.
Purpose, plan and milestones: [ROADMAP.md](ROADMAP.md) (Korean).

## Screen

Example (made-up data):

```text
 all 31n $5.2k │ prod-us 24n $4.3k │ staging 7n $0.9k │  data 14:02:11 · auto 30s · r refresh
 NODEPOOLS · prod-us · 3 pools                   ↑↓ / click: pick a pool → its nodes below · $ = list price, no SP/RI
  NODEPOOL     CONSOLIDATION   NODES→FIT   CPU             MEM             $/MO   SAVE≈/MO ▼ WHY NOT CONSOLIDATED
  (all pools)                    24 → 15   ━━━━━━────  57% ━━━━──────  41% $4,272     $1,466
▶ general      empty only        16 → 9    ━━━━━─────  52% ━━━━──────  37% $2,018       $883 16/16 non-empty consolidation disabled
  memory       underutilized      3 → 2    ━━━━━━────  61% ━━━━━━━━──  79%   $993       $331 3/3 PDB blocks eviction
  batch        underutilized      5 → 4    ━━━━━━━───  71% ━━━━━─────  48% $1,261       $252 2/5 do-not-disrupt pod

 NODES · general · 16 nodes · consolidationPolicy=WhenEmpty consolidateAfter=5m budget=10%   enter / click: pods on node
  NODE           TYPE        CAP       CPU             MEM             PODS   $/H     IDLE/MO ▼ AGE   WHY NOT CONSOLIDATED
▷ ip-10-1-34-17  m6a.xlarge  on-demand ━━────────  21% ━─────────  12%      7 $0.1728      $100    3d  non-empty consolidation disabled
  ip-10-1-52-140 m6a.xlarge  on-demand ━━━───────  33% ━━────────  18%      8 $0.1728       $85    3d  non-empty consolidation disabled
→ next ctx • ↑ up • ↓ down • tab pools⇄nodes • enter pool→nodes · node→pods • s sort • a all pools • click tab·row·header • ? more
```

## Quickstart

You need

- AWS CLI v2 and a valid session for the AWS profiles in your kubeconfig
- EKS contexts created by `aws eks update-kubeconfig`
- Go 1.26.10 or newer to install from source

```sh
go install github.com/open-dev-tools/karpenter-node-viewer/cmd/knv@latest

knv                                   # every kubeconfig context whose exec command is aws
knv -context prod-us,staging          # pick some
knv -kubeconfig ./other.kubeconfig    # default: first entry of $KUBECONFIG or ~/.kube/config; trusted files only, their exec commands run
```

Build from source: `git clone https://github.com/open-dev-tools/karpenter-node-viewer && cd karpenter-node-viewer && go build -o knv ./cmd/knv`

A context whose session expired shows `✗` in its tab. Run `aws sso login --profile <profile>` and press `r`.

## Usage

Two tables. **NODEPOOLS** has one row per NodePool (the top `(all pools)` row is the total); **NODES** lists the nodes of the pool picked above.

| To | Keys | Mouse |
|---|---|---|
| Switch context | `←` `→` (`c` `C`) | click a tab |
| Pick a NodePool → its nodes below | `↑` `↓`, `enter` to go to the nodes, `a` / `esc` for `(all pools)` | click a pool row |
| Pods on a node | `enter` in the node table, `esc` to close | click a node row, click anywhere to close |
| Sort | `s` cycles columns, `S` reverses | click a header (again to reverse) |
| Switch table / scroll | `tab`, `↑` `↓` (`j` `k`) | wheel |
| Refresh data | `r` now, every 30s automatically | |

The footer shows short help; `?` shows all of it.

| Column | Meaning |
|---|---|
| CPU · MEM | requests / allocatable, not usage |
| CONSOLIDATION | `empty only` = `WhenEmpty` (under-used nodes are never merged), `underutilized` = `WhenEmptyOrUnderutilized` |
| NODES→FIT | nodes now → nodes needed if the requests were packed perfectly onto average-sized nodes (a lower bound). Affinity, topology spread, PDBs and do-not-disrupt are not considered yet ([v0.2](ROADMAP.md#v02--추정-정확도)) |
| SAVE≈/MO | (NODES − FIT) × average node price × 730h |
| IDLE/MO | node price × (1 − max(CPU, MEM)) × 730h |
| WHY NOT CONSOLIDATED | latest message of the `Unconsolidatable` / `DisruptionBlocked` events Karpenter reported; the pool table shows the most common reason and how many nodes have it. An event's reporter fields are written by its author, not authenticated: a principal that can create Events in `default` can impersonate Karpenter |
| $ | list price; Savings Plans and RIs are not applied. Fargate nodes show `?` (their size comes from a pod annotation, so it is not trusted) |

## How it works

- Each context runs eks-node-viewer v0.7.4's `pkg/client` (node, pod and NodeClaim watches), `pkg/model` (request accounting) and `pkg/aws` (on-demand and spot prices)
- Every 30 seconds knv reads the Karpenter NodePools and the Node events in the `default` namespace; only events reported by Karpenter count as blockers
- A node's pod list is fetched from the API server when you open it
- Every 30 seconds (or on `r`) that state is frozen into a snapshot and drawn. Cursor, sort and tab changes stay inside the snapshot, so rows do not reorder between refreshes

## Minimum permissions

All read-only. Files: [`deploy/rbac.yaml`](deploy/rbac.yaml) · [`deploy/iam-policy.json`](deploy/iam-policy.json)

Kubernetes

| Resource | Verbs | Scope | Used for |
|---|---|---|---|
| nodes | list, watch | cluster | eks-node-viewer watch |
| pods | list, watch | all namespaces | eks-node-viewer watch |
| nodeclaims.karpenter.sh | list, watch | cluster | eks-node-viewer watch |
| nodepools.karpenter.sh | list | cluster | disruption settings |
| events | list | `default` namespace | consolidation blockers |

AWS IAM (neither action supports resource-level scoping, hence `"*"`)

| Action | Used for | Without it |
|---|---|---|
| `pricing:GetProducts` | on-demand prices; called with the default AWS credentials (`AWS_PROFILE` or default), not the context's profile | falls back to the static prices built into eks-node-viewer |
| `ec2:DescribeSpotPriceHistory` | spot prices; called with each context's kubeconfig `AWS_PROFILE` and `--region` | spot node prices show `?` |

To reach EKS the IAM principal must be mapped to the RBAC above, for example through an access entry.

## Dependencies

Runtime: AWS CLI v2 (called by the kubeconfig exec). kubectl is not needed.

| Go module | Version | Used for |
|---|---|---|
| github.com/awslabs/eks-node-viewer | v0.7.4 | watches, request accounting, pricing |
| github.com/charmbracelet/bubbletea | v1.3.4 | TUI loop, mouse |
| github.com/charmbracelet/bubbles | v0.21.0 | footer help, key bindings |
| github.com/charmbracelet/lipgloss | v1.1.0 | colors |
| k8s.io/client-go · api · apimachinery | v0.32.3 | NodePools and events, kubeconfig parsing |
| github.com/aws/aws-sdk-go-v2/config | v1.29.14 | per-context AWS profile and region for spot prices |
| k8s.io/klog/v2 · github.com/go-logr/logr | v2.130.1 · v1.4.2 | discards client-go logs so they do not draw over the TUI |

bubbletea, bubbles, lipgloss and client-go are the versions eks-node-viewer v0.7.4 uses. Full list: [`go.mod`](go.mod).

## Project structure

```text
.
├── cmd/knv/              binary: flags, log silencing, starting contexts
├── internal/
│   ├── cluster/          kubeconfig parsing, eks-node-viewer controllers, Karpenter polling
│   ├── snapshot/         frozen state: node and NodePool rows, fit, savings, blocker summary
│   └── tui/              bubbletea model, table rendering, keys and mouse, pods view
├── deploy/               minimum permissions: rbac.yaml, iam-policy.json
├── .github/workflows/    CI: gofmt, vet, test, build
├── ROADMAP.md            purpose, plan, milestones
├── CONTRIBUTING.md       dev setup, PR checks, RBAC verification
├── SECURITY.md           vulnerability reporting
└── LICENSE               MIT
```

## Contributing · Security · License

- Contributing: [CONTRIBUTING.md](CONTRIBUTING.md) (Korean)
- Reporting vulnerabilities: [SECURITY.md](SECURITY.md) (Korean)
- License: [MIT](LICENSE)
