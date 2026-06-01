# Zoro Xboard Node v0.1.0 - VLESS TCP HTTP Header Fix

Based on fork `nonamezr/Xboard-Node`.

## Changes

- Fix VLESS + TCP + HTTP header for sing-box runtime.
- Keep the Xray path fix for `tcpSettings.header.type=http`.
- Package Zoro installer `install-zoro.sh` for repeatable VPS installs.

## Tested

Tested with:

- VLESS
- `network=tcp`
- `header.type=http`
- Host: `www.softbank.jp`
- Path: `/`
- TLS: none
- machine mode

Fixes runtime error:

```text
unknown version: 71
```

`71` is ASCII `G` from `GET / HTTP/1.1`; the node now applies the HTTP transport/header parser instead of reading it as raw VLESS.

## Assets

- `xboard-node-linux-amd64`
- `SHA256SUMS`

## Install

Machine mode:

```bash
curl -fsSL https://raw.githubusercontent.com/nonamezr/Xboard-Node/zoro-v0.1.0/install-zoro.sh | bash -s -- --mode machine --panel "https://beta.yasuidata.com" --token "TOKEN_MOI" --machine-id MACHINE_ID
```

Node mode:

```bash
curl -fsSL https://raw.githubusercontent.com/nonamezr/Xboard-Node/zoro-v0.1.0/install-zoro.sh | bash -s -- --mode node --panel "https://beta.yasuidata.com" --token "TOKEN_MOI" --node-id NODE_ID
```
