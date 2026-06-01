# Manual GitHub release steps for zoro-v0.1.0

Use this if `gh` CLI is not installed/authenticated.

## 1. Push branch and tag

From `/opt/xboard-node-dev`:

```bash
git push origin fix/vless-tcp-http-header
git push origin zoro-v0.1.0
```

## 2. Create release on GitHub

Open:

```text
https://github.com/nonamezr/Xboard-Node/releases/new
```

Fields:

- Tag: `zoro-v0.1.0`
- Target: `fix/vless-tcp-http-header` or the pushed tag commit
- Title: `Zoro Xboard Node v0.1.0 - VLESS TCP HTTP Header Fix`
- Notes: paste content from `RELEASE_NOTES_ZORO_V0.1.0.md`

Upload assets:

```text
/opt/xboard-node-dev/release/xboard-node-linux-amd64
/opt/xboard-node-dev/release/SHA256SUMS
```

Publish release.

## 3. Verify asset URLs

```bash
curl -I https://github.com/nonamezr/Xboard-Node/releases/download/zoro-v0.1.0/xboard-node-linux-amd64
curl -fsSL https://github.com/nonamezr/Xboard-Node/releases/download/zoro-v0.1.0/SHA256SUMS
```

## 4. Verify installer raw URL

```bash
curl -I https://raw.githubusercontent.com/nonamezr/Xboard-Node/zoro-v0.1.0/install-zoro.sh
```
