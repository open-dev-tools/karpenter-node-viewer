# 보안 정책

## 취약점 신고

공개 이슈로 올리지 말고 GitHub 비공개 신고를 쓴다 — 이 리포의 **Security** 탭 → **Report a vulnerability**.

## knv 가 다루는 것

- 읽기 전용이다. 클러스터와 AWS 계정에 쓰지 않는다. 필요한 권한은 [deploy/](deploy/) 와 README 의 최소 권한 표에 있다
- knv 자체는 자격증명을 저장하지 않는다. 인증은 kubeconfig(exec 플러그인 등)와 AWS SDK 기본 자격증명 체인에 맡긴다. 이 라이브러리들은 사용자 파일에 쓸 수 있다 — client-go 의 kubeconfig 마이그레이션(`~/.kube/.kubeconfig` → `~/.kube/config`)과 oidc 토큰 갱신, AWS SDK 의 SSO 토큰 캐시
- knv 가 직접 보내는 요청은 kube API server 와 AWS API(Pricing · EC2 spot 가격)뿐이다. 그 밖에 kubeconfig 와 AWS 프로파일이 정한 곳(exec 플러그인, `credential_process`, SSO, oidc issuer, 인스턴스 메타데이터)으로 인증 요청이 나갈 수 있다
- 신뢰하는 kubeconfig 만 쓴다. knv 는 선택한 컨텍스트마다 exec 명령을 실행하고 그 `server` 로 토큰을 보낸다. 기본으로 선택되는 것은 exec 명령이 `aws` 인 컨텍스트 전부다
- 파드를 모든 namespace 에서 watch 하므로 파드 spec(인라인 env 값 포함)이 프로세스 메모리에 올라온다. 화면에는 이름과 요청량만 나온다
- 클러스터에서 온 문자열(이벤트 메시지, 노드 · 파드 이름, API 오류)은 제어 문자를 `?` 로 바꿔 그리고, consolidation 차단 사유는 Karpenter 가 보고한 이벤트만 쓴다
- 화면에 노드 · 파드 · 네임스페이스 이름이 나온다. 화면을 공유할 때 주의한다
