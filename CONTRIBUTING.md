# 기여하기

## 개발 환경

- Go 1.24.2 이상
- kubeconfig 에 EKS 컨텍스트 하나 이상. 없으면 [kind](https://kind.sigs.k8s.io/) 로 화면만 확인할 수 있다 (가격 · Karpenter 정보는 비어 있다)

```sh
git clone https://github.com/open-dev-tools/karpenter-node-viewer
cd karpenter-node-viewer
go run ./cmd/knv -context <컨텍스트>
```

## 코드 구조

데이터는 한 방향으로 흐른다: `internal/cluster` (watch · 폴링) → `internal/snapshot` (시점 고정 · 집계) → `internal/tui` (그리기 · 입력).

- 클러스터를 읽는 코드는 `internal/cluster` 에만 둔다
- 숫자 계산(FIT · 절감 · 차단 사유 요약)은 `internal/snapshot` 에 두고 테스트를 붙인다
- 커서 · 정렬 · 탭 조작으로 클러스터를 다시 읽지 않는다. `Model.derive` 는 마지막 스냅샷만 본다

처음 커밋부터 순서대로 읽으면 구조가 쌓인 순서가 보인다.

```sh
git log --reverse --stat
```

## PR 전에

```sh
gofmt -l .          # 출력이 없어야 한다
go vet ./...
go test ./...
go build ./cmd/knv
```

CI 가 같은 것을 돌린다.

- PR 하나에 목적 하나. [ROADMAP](ROADMAP.md) 의 이슈를 링크한다
- 커밋 메시지는 영어 명령형 한 줄과, 왜 바꿨는지 쓴 본문
- Kubernetes · AWS 호출을 추가하면 `deploy/rbac.yaml` · `deploy/iam-policy.json` 과 README 의 권한 표를 같이 고친다

## RBAC 을 바꿨다면 — kind 로 확인

`deploy/rbac.yaml` 만 묶은 ServiceAccount 로 knv 를 띄워 본다.

```sh
kind create cluster --name knv-rbac
kubectl --context kind-knv-rbac apply -f deploy/rbac.yaml
kubectl --context kind-knv-rbac create sa knv
kubectl --context kind-knv-rbac create clusterrolebinding knv-viewer --clusterrole=knv-viewer --serviceaccount=default:knv
kubectl --context kind-knv-rbac create rolebinding knv-events -n default --role=knv-events --serviceaccount=default:knv

# 토큰만 쓰는 kubeconfig. 남이 못 읽는 임시 파일(mode 600)에 만들고, admin 사용자는 쓰기 전에 지운다
# (클라이언트 인증서가 남아 있으면 admin 으로 인증된다)
KCFG="$(umask 077; mktemp)"
kubectl config view --raw --minify --flatten --context kind-knv-rbac > "$KCFG"
kubectl --kubeconfig "$KCFG" config set-credentials knv --token="$(kubectl --context kind-knv-rbac create token knv)"
kubectl --kubeconfig "$KCFG" config set-context knv --cluster=kind-knv-rbac --user=knv
kubectl --kubeconfig "$KCFG" config use-context knv
kubectl --kubeconfig "$KCFG" config delete-context kind-knv-rbac
kubectl --kubeconfig "$KCFG" config delete-user kind-knv-rbac

kubectl --kubeconfig "$KCFG" get --raw /api/v1/nodes > /dev/null && echo allowed
kubectl --kubeconfig "$KCFG" get --raw /api/v1/secrets       # Forbidden 이어야 한다
go run ./cmd/knv -kubeconfig "$KCFG" -context knv            # ✗ 없이 떠야 한다

rm -f "$KCFG"; kind delete cluster --name knv-rbac
```

Karpenter CRD 가 없는 kind 에서 `karpenter.sh` 경로는 인가를 통과하면 `NotFound`, 막히면 `Forbidden` 이다.
