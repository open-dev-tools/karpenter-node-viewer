# 보안 정책

## 취약점 신고

공개 이슈로 올리지 말고 GitHub 비공개 신고를 쓴다 — 이 리포의 **Security** 탭 → **Report a vulnerability**.

## knv 가 다루는 것

- 읽기 전용이다. 클러스터와 AWS 계정에 쓰지 않는다. 필요한 권한은 [deploy/](deploy/) 와 README 의 최소 권한 표에 있다
- 자격증명을 저장하거나 따로 전송하지 않는다. kubeconfig 의 exec(`aws eks get-token`)와 AWS SDK 기본 자격증명 체인을 그대로 쓴다
- 나가는 연결은 kube API server 와 AWS API(Pricing · EC2 spot 가격, SSO 프로파일이면 자격증명 갱신)뿐이다
- 화면에 노드 · 파드 · 네임스페이스 이름이 나온다. 화면을 공유할 때 주의한다
