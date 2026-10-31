# karpenter-node-viewer (`knv`)

여러 EKS 컨텍스트의 Karpenter NodePool 을 한 터미널 화면에서 본다 — 얼마나 덜 찼는지, 몇 대면 되는지, 무엇이 consolidation 을 막는지.

![Go](https://img.shields.io/badge/Go-1.24+-00ADD8)
![License](https://img.shields.io/badge/License-MIT-yellow)

**한국어** | [English](README_EN.md)

## 왜

[eks-node-viewer](https://github.com/awslabs/eks-node-viewer) 는 노드를 잘 보여주지만 컨텍스트 하나씩, 노드 단위로만 보여주고 Karpenter 가 왜 노드를 줄이지 않는지는 보여주지 않는다.
knv 는 eks-node-viewer 의 데이터 층(watch · 요청 합산 · 가격)을 그대로 쓰고, 그 위에 NodePool 단위 요약과 consolidation 차단 사유를 얹었다.
목적 · 계획 · 마일스톤은 [ROADMAP.md](ROADMAP.md).

## 화면

예시 (가상 데이터):

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

## 퀵스타트

필요한 것

- `aws` CLI v2 와, kubeconfig 에 적힌 AWS 프로파일의 유효한 세션
- `aws eks update-kubeconfig` 로 만든 EKS 컨텍스트
- 소스로 설치할 때 Go 1.24.2 이상

```sh
go install github.com/open-dev-tools/karpenter-node-viewer/cmd/knv@latest

knv                                   # kubeconfig 에서 exec 명령이 aws 인 컨텍스트 전부
knv -context prod-us,staging          # 골라서
knv -kubeconfig ./other.kubeconfig    # 기본값은 $KUBECONFIG 첫 항목 또는 ~/.kube/config. 신뢰하는 파일만 — exec 명령이 실행된다
```

소스에서 빌드: `git clone https://github.com/open-dev-tools/karpenter-node-viewer && cd karpenter-node-viewer && go build -o knv ./cmd/knv`

세션이 만료된 컨텍스트는 탭에 `✗` 로 보인다. `aws sso login --profile <프로파일>` 후 `r`.

## 사용법

화면은 표 두 개다. 위 **NODEPOOLS** 는 NodePool 당 한 줄(맨 위 `(all pools)` = 합계), 아래 **NODES** 는 위에서 고른 NodePool 의 노드다.

| 하려는 것 | 키 | 마우스 |
|---|---|---|
| 컨텍스트 바꾸기 | `←` `→` (`c` `C`) | 맨 위 탭 클릭 |
| NodePool 고르기 → 아래에 그 노드 | `↑` `↓`, `enter` 로 노드 표로, `a` · `esc` 로 `(all pools)` | NodePool 행 클릭 |
| 노드의 파드 목록 | 노드 표에서 `enter`, 닫기 `esc` | 노드 행 클릭, 아무 데나 클릭해 닫기 |
| 정렬 | `s` 열 순환, `S` 역순 | 헤더 클릭 (다시 클릭하면 역순) |
| 표 전환 · 스크롤 | `tab`, `↑` `↓` (`j` `k`) | 휠 |
| 데이터 갱신 | `r` 즉시, 자동 30초 | |

footer 에 짧은 도움말, `?` 로 전체 도움말.

| 열 | 뜻 |
|---|---|
| CPU · MEM | 요청(requests) / allocatable. 실사용량이 아니다 |
| CONSOLIDATION | `empty only` = `WhenEmpty` (덜 찬 노드를 합치지 않는다), `underutilized` = `WhenEmptyOrUnderutilized` |
| NODES→FIT | 지금 노드 수 → 요청을 평균 크기 노드에 빈틈없이 채웠을 때 필요한 수 (하한). affinity · topology spread · PDB · do-not-disrupt 는 아직 보지 않는다 ([v0.2](ROADMAP.md#v02--추정-정확도)) |
| SAVE≈/MO | (NODES − FIT) × 평균 노드 가격 × 730h |
| IDLE/MO | 노드 가격 × (1 − max(CPU, MEM)) × 730h |
| WHY NOT CONSOLIDATED | Karpenter `Unconsolidatable` · `DisruptionBlocked` 이벤트의 최근 메시지. NodePool 표는 가장 많은 사유와 노드 수 |
| $ | list price. Savings Plans · RI 미반영. Fargate 노드는 `?` (파드 annotation 으로 정해지는 값이라 쓰지 않는다) |

## 동작 방식

- 컨텍스트마다 eks-node-viewer v0.7.4 의 `pkg/client`(node · pod · NodeClaim watch) · `pkg/model`(요청 합산) · `pkg/aws`(On-Demand · Spot 가격)를 하나씩 띄운다
- 30초마다 Karpenter NodePool 과 `default` 네임스페이스의 Node 이벤트를 읽는다. 차단 사유는 Karpenter 가 보고한 이벤트만 쓴다
- 노드의 파드 목록은 열 때 API 서버에서 그 노드의 파드를 조회한다
- 30초마다(또는 `r`) 이 상태를 스냅샷으로 고정해 그린다. 커서 · 정렬 · 탭은 스냅샷 안에서만 움직이므로 갱신 사이에 행 순서가 바뀌지 않는다

## 최소 권한

전부 읽기다. 파일: [`deploy/rbac.yaml`](deploy/rbac.yaml) · [`deploy/iam-policy.json`](deploy/iam-policy.json)

Kubernetes

| 리소스 | verb | 범위 | 쓰는 곳 |
|---|---|---|---|
| nodes | list, watch | 클러스터 | eks-node-viewer watch |
| pods | list, watch | 모든 namespace | eks-node-viewer watch |
| nodeclaims.karpenter.sh | list, watch | 클러스터 | eks-node-viewer watch |
| nodepools.karpenter.sh | list | 클러스터 | disruption 설정 |
| events | list | `default` namespace | consolidation 차단 사유 |

AWS IAM (두 action 모두 resource 수준 제한이 없어 `"*"`)

| action | 쓰는 곳 | 없으면 |
|---|---|---|
| `pricing:GetProducts` | On-Demand 가격. 컨텍스트 프로파일이 아니라 기본 AWS 자격증명(`AWS_PROFILE` 또는 default)으로 호출된다 | eks-node-viewer 에 내장된 정적 가격을 쓴다 |
| `ec2:DescribeSpotPriceHistory` | Spot 가격. 각 컨텍스트의 kubeconfig `AWS_PROFILE` · `--region` 으로 호출된다 | Spot 노드 가격이 `?` |

EKS 에 접근하려면 IAM principal 이 access entry 등으로 위 RBAC 에 매핑돼 있어야 한다.

## 의존성

실행: `aws` CLI v2 (kubeconfig exec 가 부른다). kubectl 은 필요 없다.

| Go 모듈 | 버전 | 쓰는 곳 |
|---|---|---|
| github.com/awslabs/eks-node-viewer | v0.7.4 | watch · 요청 합산 · 가격 |
| github.com/charmbracelet/bubbletea | v1.3.4 | TUI 루프 · 마우스 |
| github.com/charmbracelet/bubbles | v0.21.0 | footer 도움말 · 키 바인딩 |
| github.com/charmbracelet/lipgloss | v1.1.0 | 색 |
| k8s.io/client-go · api · apimachinery | v0.32.3 | NodePool · 이벤트 조회, kubeconfig 해석 |
| github.com/aws/aws-sdk-go-v2/config | v1.29.14 | Spot 가격 조회용 컨텍스트별 프로파일 · 리전 |
| k8s.io/klog/v2 · github.com/go-logr/logr | v2.130.1 · v1.4.2 | client-go 로그를 버린다 (TUI 깨짐 방지) |

bubbletea · bubbles · lipgloss · client-go 버전은 eks-node-viewer v0.7.4 와 같다. 전체 목록은 [`go.mod`](go.mod).

## 폴더 구조

```text
.
├── cmd/knv/              실행 파일: 플래그, 로그 차단, 컨텍스트 시작
├── internal/
│   ├── cluster/          kubeconfig 해석, eks-node-viewer controller, Karpenter 폴링
│   ├── snapshot/         시점 고정: 노드 · NodePool 집계, FIT · 절감 · 차단 사유 요약
│   └── tui/              bubbletea 모델, 표 그리기, 키 · 마우스, 파드 목록
├── deploy/               최소 권한: rbac.yaml, iam-policy.json
├── .github/workflows/    CI: gofmt, vet, test, build
├── ROADMAP.md            목적 · 계획 · 마일스톤
├── CONTRIBUTING.md       개발 환경, PR 체크, RBAC 검증
├── SECURITY.md           취약점 신고
└── LICENSE               MIT
```

## 기여 · 보안 · 라이선스

- 기여: [CONTRIBUTING.md](CONTRIBUTING.md)
- 취약점 신고: [SECURITY.md](SECURITY.md)
- 라이선스: [MIT](LICENSE)
