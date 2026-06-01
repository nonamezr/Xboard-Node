# Zoro Xboard Node release report

## 1. Branch / tag / release

- Local repo: `/opt/xboard-node-dev`
- GitHub fork: `https://github.com/nonamezr/Xboard-Node`
- Branch: `fix/vless-tcp-http-header`
- Local packaging commit: `8a7d5bd Package Zoro install script for VLESS TCP HTTP header fix`
- Local annotated tag: `zoro-v0.1.0`
- Release title: `Zoro Xboard Node v0.1.0 - VLESS TCP HTTP Header Fix`

Push/release status:

- Push branch: not completed, GitHub HTTPS auth unavailable on this host.
- Push tag: not completed, GitHub HTTPS auth unavailable on this host.
- `gh` CLI: not installed, so GitHub Release was not created automatically.
- Manual release steps are in `GITHUB-RELEASE-MANUAL-STEPS.md`.

## 2. Commit chứa fix

Fix commits:

- `e7efe53` — Xray path: add `tcpSettings.header.type=http` for VLESS TCP HTTP header.
- `6341926` — sing-box path: add V2Ray HTTP transport for VLESS TCP HTTP header.
- `8a7d5bd` — package installer/release assets.

Latest local tag `zoro-v0.1.0` points at packaging commit `8a7d5bd`.

## 3. Binary release

Release asset path:

```text
/opt/xboard-node-dev/release/xboard-node-linux-amd64
```

Copied from:

```text
/opt/xboard-node-dev/build/xboard-node-custom
```

## 4. SHA256

```text
3802c9a44cdda5978c458ca723f1bc685e540bd24d1a585fe068e0b8d6b07814  xboard-node-linux-amd64
```

Checksum file:

```text
/opt/xboard-node-dev/release/SHA256SUMS
```

## 5. Installer

Installer path:

```text
/opt/xboard-node-dev/install-zoro.sh
```

Defaults:

- Release base: `https://github.com/nonamezr/Xboard-Node/releases`
- Version: `zoro-v0.1.0`
- Binary URL: `https://github.com/nonamezr/Xboard-Node/releases/download/zoro-v0.1.0/xboard-node-linux-amd64`

`install-zoro.sh` is based on upstream installer and keeps support for:

- `--mode node`
- `--mode machine`
- `--panel`
- `--token`
- `--node-id`
- `--machine-id`

It backs up existing binary/config/service under `/etc/xboard-node/backups/YYYYMMDD-HHMMSS`, installs systemd service, runs health checks, and attempts rollback on install failure.

## 6. Lệnh cài VPS mới

Machine mode:

```bash
curl -fsSL https://raw.githubusercontent.com/nonamezr/Xboard-Node/zoro-v0.1.0/install-zoro.sh | bash -s -- --mode machine --panel "https://beta.yasuidata.com" --token "TOKEN_MOI" --machine-id MACHINE_ID
```

Node mode:

```bash
curl -fsSL https://raw.githubusercontent.com/nonamezr/Xboard-Node/zoro-v0.1.0/install-zoro.sh | bash -s -- --mode node --panel "https://beta.yasuidata.com" --token "TOKEN_MOI" --node-id NODE_ID
```

More commands are in:

```text
/opt/xboard-node-dev/ZORO-INSTALL-COMMANDS.md
```

## 7. Tests đã pass

Source tests:

```text
go test ./...: PASS
```

Build:

```text
build linux amd64: PASS
```

Runtime test reported by Zoro:

- machine mode: OK
- VLESS tcp http header: OK
- Host: `www.softbank.jp`
- Path: `/`
- TLS: none
- `unknown version: 71`: fixed

## 8. Rollback nếu cài lỗi

Installer tự backup vào:

```text
/etc/xboard-node/backups/YYYYMMDD-HHMMSS
```

Manual rollback example:

```bash
sudo systemctl stop xboard-node
sudo install -m 755 /etc/xboard-node/backups/YYYYMMDD-HHMMSS/xboard-node /usr/local/bin/xboard-node
sudo install -m 600 /etc/xboard-node/backups/YYYYMMDD-HHMMSS/config.yml /etc/xboard-node/config.yml
sudo install -m 600 /etc/xboard-node/backups/YYYYMMDD-HHMMSS/credentials.env /etc/xboard-node/credentials.env
sudo systemctl daemon-reload
sudo systemctl start xboard-node
journalctl -u xboard-node -n 120 --no-pager
```

Do not delete backup.

## 9. Zoro cần làm sau release

Because GitHub auth is unavailable here, Zoro needs to push/tag/release manually or provide auth:

```bash
cd /opt/xboard-node-dev
git push origin fix/vless-tcp-http-header
git push origin zoro-v0.1.0
```

Then create GitHub Release using `GITHUB-RELEASE-MANUAL-STEPS.md`, upload:

```text
release/xboard-node-linux-amd64
release/SHA256SUMS
```

Security reminders:

- Regenerate token for new VPS.
- Use a fresh token in install commands.
- Do not reuse any old/pasted token.
- Do not commit tokens to repo.

## 10. Notes

No Horivex touched. No production touched. No panel `/opt/xboard-origin` touched. No binary on running node changed during packaging.
