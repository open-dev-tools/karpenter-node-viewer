# 로드맵

## 목적

Karpenter 를 쓰는 EKS 클러스터에서 노드 비용이 새는 곳은 대개 NodePool 단위다. consolidation 정책이 덜 찬 노드를 합치지 않거나, PDB · do-not-disrupt 파드가 노드를 붙잡는다.
[eks-node-viewer](https://github.com/awslabs/eks-node-viewer) 는 노드를 잘 보여주지만 컨텍스트 하나씩, 노드 단위로만 보여주고, Karpenter 가 왜 노드를 줄이지 않는지는 보여주지 않는다.

knv 는 한 화면에서 세 가지에 답한다.

1. 어느 NodePool 이 가장 많이 남기고 있나 — 요청 기준 사용률과 월 비용
2. 빈틈없이 채우면 몇 대면 되나 — `NODES→FIT` 와 `SAVE≈/MO`
3. 무엇이 consolidation 을 막고 있나 — Karpenter `Unconsolidatable` · `DisruptionBlocked` 이벤트

## 하지 않는 것

- 클러스터를 바꾸지 않는다. 읽기 권한만 쓴다
- 청구서를 대신하지 않는다. 금액은 list price 이고 Savings Plans · RI 를 반영하지 않는다
- Karpenter 지표 · 대시보드(Prometheus · Grafana)를 대신하지 않는다. 지금 상태를 터미널에서 보는 도구다

## 마일스톤

### v0.1 — 한 화면에서 보기 (완료)

- [x] kubeconfig 의 EKS 컨텍스트 여러 개를 동시에 watch (eks-node-viewer 패키지 재사용)
- [x] NodePool disruption 설정과 consolidation 차단 이벤트 수집
- [x] NodePool 별 요청 사용률 · 월 비용 · FIT · 절감 추정 · 차단 사유 요약
- [x] 노드의 파드 목록
- [x] 키보드 · 마우스 · footer 도움말, 30초 주기 갱신
- [x] 최소 RBAC · IAM 문서화

### v0.2 — 추정 정확도

`FIT` 는 지금 하한이다. 노드 하나에 파드 하나만 뜨도록 묶인 NodePool 에서는 닿을 수 없는 절감을 보여준다.

- [ ] required podAntiAffinity(hostname) · topologySpreadConstraints 를 FIT 에 반영하고, 반영 못 하는 NodePool 은 표시
- [ ] DaemonSet 요청을 노드 용량에서 빼고 계산
- [ ] requests 가 없는 파드(BestEffort) 표시 — 사용률이 실제보다 낮아 보이는 원인
- [ ] metrics-server 사용량을 함께 보여 requests 과다 할당(right-sizing 여지) 표시

### v0.3 — 내보내기

- [ ] `--once --json` 스냅샷 출력 (스크립트 · 리뷰 자동화용)
- [ ] TTY 없는 환경에서 표를 한 번 출력

### v0.4 — 배포

- [ ] GoReleaser 로 macOS · Linux 바이너리와 체크섬
- [ ] Homebrew tap
- [ ] 바이너리용 서드파티 라이선스 고지(ATTRIBUTION)
- [ ] `--version`

### v1.0 — 안정화

- [ ] 플래그 · 출력 형식 고정
- [ ] TUI 동작 테스트 (teatest)
- [ ] `aws eks get-token` 이 아닌 인증을 쓰는 컨텍스트의 리전 · 가격 조회 경로 정리

## 계획

- 마일스톤은 위 순서대로 진행한다. 정확도(v0.2)가 배포(v0.4)보다 먼저다 — 틀린 절감액을 널리 배포하지 않는다
- 항목 하나 = GitHub 이슈 하나 = PR 하나. 이슈에 마일스톤을 붙인다
- 큰 설계 변경(예: 파드 spec 을 직접 watch 하기)은 이슈에서 먼저 합의한다
