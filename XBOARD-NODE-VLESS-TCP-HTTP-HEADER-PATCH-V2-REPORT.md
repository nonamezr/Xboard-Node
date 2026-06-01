# Xboard Node VLESS TCP HTTP header patch V2 report

## 1. Vì sao patch V1 chưa hết lỗi

Patch V1 chỉ sửa Xray config builder tại `internal/kernel/xray/config.go` để sinh `streamSettings.tcpSettings.header`.

Runtime test thật vẫn log:

```text
inbound/vless[vless-in]: unknown version: 71
```

Format log này nghiêng về sing-box inbound. Với sing-box, V1 không có tác dụng vì `internal/kernel/singbox/config.go` vẫn bỏ qua transport khi `network=tcp`, nên inbound tiếp tục đọc raw VLESS và parse byte đầu `G` của `GET / HTTP/1.1` thành VLESS version.

## 2. Runtime thật là xray hay sing-box

Trong source:

- Kernel được chọn ở `internal/service/service.go:newService()` theo `cfg.Kernel.Type`.
- Default kernel là `singbox` trong `internal/config/config.go:setDefaultsFrom()`.
- Machine mode có auto-switch ở `internal/machine/machine.go`, nhưng chỉ switch sang xray với transport không support bởi sing-box như `xhttp/splithttp`.
- `tcp` không bị auto-switch, nên node test mặc định chạy sing-box nếu config/panel không override.
- Panel/config có thể chọn kernel bằng `kernel.type` local config hoặc `kernel_type` từ panel node config.

Kết luận: runtime thật cho node test khả năng cao là sing-box. Patch V1 đúng là chỉ ảnh hưởng Xray path.

## 3. File sửa lần này

- `internal/kernel/singbox/config.go`
- `internal/kernel/singbox/config_test.go`
- `XBOARD-NODE-VLESS-TCP-HTTP-HEADER-PATCH-V2-REPORT.md`

Không sửa Horivex, production, panel `/opt/xboard-origin`, secrets, hoặc binary đang chạy.

## 4. Config output trước/sau

### Trước V2: VLESS tcp http trên sing-box

`applyTransport()` trả về ngay khi `network=tcp`, nên inbound không có transport:

```json
{
  "type": "vless",
  "tag": "vless-in",
  "listen": "::",
  "listen_port": 443,
  "users": ["..."]
}
```

Kết quả: sing-box đọc `GET / HTTP/1.1` như raw VLESS và log `unknown version: 71`.

### Sau V2: VLESS tcp http trên sing-box

Khi:

- protocol = `vless`
- network = `tcp`
- header.type = `http`
- host = `www.softbank.jp`
- path = `/`

Inbound sing-box sinh thêm V2Ray HTTP transport:

```json
{
  "type": "vless",
  "tag": "vless-in",
  "listen": "::",
  "listen_port": 443,
  "transport": {
    "type": "http",
    "host": ["www.softbank.jp"],
    "path": "/",
    "headers": {
      "Host": ["www.softbank.jp"]
    }
  }
}
```

Schema được đối chiếu từ sing-box trong dependency hiện tại: `docs/configuration/shared/v2ray-transport.md` ghi rõ sing-box không có TCP transport riêng; plain HTTP được merge vào transport `type=http`, TLS không bắt buộc nên TLS none dùng HTTP/1.1 plain.

## 5. Điều kiện áp dụng / phạm vi ảnh hưởng

Chỉ áp dụng khi:

- `nc.Protocol == "vless"`
- `nc.Network == "tcp"`
- header type là `http`

Header type nhận các shape phổ biến:

- `header.type`
- `headerType`
- `header_type`
- `type`
- `header = "http"`
- nested `header.type`

Normalize:

- Host string -> `[]string{"host"}`; array giữ array.
- Path string -> string path đầu tiên cho sing-box `transport.path`.
- Path rỗng -> fallback `/`.

Không đổi:

- VLESS tcp none: không có transport.
- VLESS ws/grpc/reality/tls: giữ logic cũ.
- vmess/trojan/shadowsocks/hysteria/tuic: không đụng.
- Xray V1 vẫn giữ nguyên.

## 6. Tests

Đã thêm sing-box tests:

- `TestBuildInbound_VLESS_TCPHTTPHeader`
- `TestBuildInbound_VLESS_TCPNoneHasNoTransport`
- `TestBuildInbound_VLESS_WSUnchangedByHTTPHeader`

Kết quả:

```text
go test ./...: PASS
```

Command dùng:

```bash
docker run --rm -v /opt/xboard-node-dev:/src -w /src golang:1.26 sh -c 'gofmt -w internal/kernel/singbox/config.go internal/kernel/singbox/config_test.go && go test ./...'
```

Máy host không có native `go`, nên dùng Docker `golang:1.26`.

## 7. Build

Kết quả:

```text
build: PASS
```

Binary mới:

```text
/opt/xboard-node-dev/build/xboard-node-custom
```

Build command:

```bash
docker run --rm -v /opt/xboard-node-dev:/src -w /src golang:1.26 sh -c 'go build -ldflags "-s -w -X main.version=... -X main.buildTime=... -X main.commit=..." -tags "with_quic with_utls with_wireguard with_clash_api" -o build/xboard-node-custom ./cmd/xboard-node'
```

## 8. Cách đem binary mới sang node test

Chỉ dùng node test, không production.

```bash
scp /opt/xboard-node-dev/build/xboard-node-custom root@45.76.48.225:/root/xboard-node-custom-v2
```

Trên node test:

```bash
sudo cp -a /usr/local/bin/xboard-node /root/xboard-node.bak.$(date +%Y%m%d-%H%M%S)
sudo systemctl stop xboard-node
sudo install -m 0755 /root/xboard-node-custom-v2 /usr/local/bin/xboard-node
sudo systemctl start xboard-node
journalctl -u xboard-node -f
```

Test lại VLESS tcp http header:

- Host: `www.softbank.jp`
- Path: `/`
- TLS: none

Kỳ vọng: không còn `unknown version: 71`.

Rollback:

```bash
sudo systemctl stop xboard-node
sudo install -m 0755 /root/xboard-node.bak.YYYYMMDD-HHMMSS /usr/local/bin/xboard-node
sudo systemctl start xboard-node
journalctl -u xboard-node -f
```

## 9. Kết luận sing-box support

sing-box dependency hiện tại có support V2Ray HTTP transport, và tài liệu nói plain HTTP không cần TLS. Vì vậy không cần hack bừa; mapping đúng là `transport.type=http` cho case Xray `tcpSettings.header.type=http`.

## 10. Kết luận

V2 patch đã sửa đúng sing-box path. `go test ./...` pass, build pass. Binary mới nằm ở:

```text
/opt/xboard-node-dev/build/xboard-node-custom
```

Có thể copy binary V2 sang node test để thử lại.
