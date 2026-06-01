# Xboard Node VLESS TCP HTTP header patch report

## 1. Repo / branch / commit

- Repo: https://github.com/nonamezr/Xboard-Node
- Local path: /opt/xboard-node-dev
- Branch: fix/vless-tcp-http-header
- Base commit before local patch: 0a29338e1f102a462363ce3527417029f89bab28
- Go version used for test/build: go1.26.3 linux/amd64 via Docker image golang:1.26
- Local host has no native go binary in PATH, so test/build used Docker only.

## 2. Files changed

- internal/kernel/xray/config.go
- internal/kernel/xray/config_test.go
- XBOARD-NODE-CUSTOM-TEST-RUNBOOK.md
- XBOARD-NODE-VLESS-TCP-HTTP-HEADER-PATCH-REPORT.md

## 3. Cause of `unknown version: 71`

71 is ASCII `G`. The client sends HTTP camouflage first, for example `GET / HTTP/1.1`. The Xray inbound was built as raw VLESS TCP without `tcpSettings.header.type=http`, so Xray tried to parse the first byte `G` as the VLESS protocol version and logged `unknown version: 71`.

## 4. Source gap / mismatch

The panel node config is carried through `panel.NodeConfig.NetworkSettings` into `model.NodeSpec.NetworkSettings`. Xray inbound generation happens in `internal/kernel/xray/config.go` via `buildVLESS -> applyStreamSettings`.

Before this patch, `applyStreamSettings` had a `case "tcp"` branch with no extra settings, so `header.type=http`, host, and path from `networkSettings` were not converted into Xray `tcpSettings.header`.

Sing-box path was reviewed too: `internal/kernel/singbox/config.go` skips transport for empty/tcp network. The target runtime config requested here is Xray-style `tcpSettings`; this patch does not hack sing-box.

## 5. Fix added

Only for:

- protocol = `vless`
- network = `tcp`
- header type = `http`

The Xray streamSettings now includes:

```json
{
  "network": "tcp",
  "tcpSettings": {
    "header": {
      "type": "http",
      "request": {
        "path": ["/"],
        "headers": { "Host": ["www.softbank.jp"] }
      }
    }
  }
}
```

Compatibility details:

- Header type accepts common panel shapes: `header.type`, `headerType`, `header_type`, `type`, `header="http"`, or nested `header.type`.
- Host string becomes `[]string{host}`; existing arrays are preserved.
- Path string becomes `[]string{path}`; existing arrays are preserved.
- Empty path with HTTP header falls back to `/`.
- TCP header none/empty remains unchanged.
- Non-VLESS and non-TCP networks are untouched.

## 6. `go test ./...`

PASS.

Command used:

```bash
docker run --rm -v /opt/xboard-node-dev:/src -w /src golang:1.26 sh -c 'gofmt -w internal/kernel/xray/config.go internal/kernel/xray/config_test.go && go test ./...'
```

## 7. Build

PASS.

Binary built at:

```text
/opt/xboard-node-dev/build/xboard-node-custom
```

Build command used:

```bash
docker run --rm -v /opt/xboard-node-dev:/src -w /src golang:1.26 sh -c 'go build -ldflags "-s -w -X main.version=... -X main.buildTime=... -X main.commit=..." -tags "with_quic with_utls with_wireguard with_clash_api" -o build/xboard-node-custom ./cmd/xboard-node'
```

## 8. Unit/config tests

Added tests:

- `TestBuildConfig_VLESS_TCPHTTPHeader`: validates `tcpSettings.header.type=http`, path `["/"]`, Host `["www.softbank.jp"]`.
- `TestBuildConfig_VLESS_TCPNoneHasNoHTTPHeader`: validates tcp none does not get HTTP header.
- `TestBuildConfig_VLESS_WSUnchangedByHTTPHeader`: validates ws path is not converted to tcpSettings.

All pass under `go test ./...`.

## 9. Protocol impact

Expected impact is limited to VLESS + tcp + http header in the Xray config builder. No behavior change intended for:

- VLESS tcp none
- VLESS ws/grpc/reality/tls
- vmess
- trojan
- shadowsocks
- hysteria
- tuic
- sing-box runtime

## 10. Test on real node

Use the runbook:

```text
/opt/xboard-node-dev/XBOARD-NODE-CUSTOM-TEST-RUNBOOK.md
```

Test on a non-production node only. Do not overwrite production binary without approval.

## 11. Rollback

Rollback steps are included in `XBOARD-NODE-CUSTOM-TEST-RUNBOOK.md`: stop service, restore backed up binary/config, start service, follow logs.

## 12. PR upstream?

Yes, this is a small targeted compatibility fix and is suitable for a PR after Zoro confirms runtime test on a test node. Do not push/PR until approved.

## 13. Conclusion

Build/test pass. Có thể đem binary custom sang node test để thử.
