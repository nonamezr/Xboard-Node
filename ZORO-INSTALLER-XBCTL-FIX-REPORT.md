# Zoro installer xbctl fix report

## Problem

Release `zoro-v0.1.0` only ships:

- `xboard-node-linux-amd64`
- `SHA256SUMS`

But `install-zoro.sh` tried to download `xbctl-linux-amd64` from the Zoro/nonamezr release URL:

```text
https://github.com/nonamezr/Xboard-Node/releases/download/zoro-v0.1.0/xbctl-linux-amd64
```

That asset does not exist, causing 404. The installer then incorrectly fell back to copying the `xboard-node` binary as `xbctl`, which later failed the `xbctl version` check.

## Fix

Changed `install-zoro.sh` so:

- `xboard-node` still downloads from Zoro release:

```text
https://github.com/nonamezr/Xboard-Node/releases/download/zoro-v0.1.0/xboard-node-linux-amd64
```

- `xbctl` downloads from official upstream latest release:

```text
https://github.com/cedar2025/xboard-node/releases/latest/download/xbctl-linux-amd64
```

- Removed fallback that used `xboard-node` as `xbctl`.
- If official `xbctl` download fails, the installer now exits with a clear error.

## File changed

```text
install-zoro.sh
```

## Validation

Syntax check:

```text
bash -n install-zoro.sh: OK
```

Asset URL checks:

```text
Zoro xboard-node asset: HTTP 302 -> release asset OK
Official xbctl asset: HTTP 302 -> v1.13 asset OK
```

Downloaded official xbctl and verified:

```text
xbctl v1.13 (built 2026-04-19T20:10:46Z)
```

## Commit / tag

Local commit:

```text
70aa87b Fix Zoro installer xbctl download source
```

Local tag created:

```text
zoro-v0.1.1
```

## Release recommendation

Create GitHub release `zoro-v0.1.1` and include:

- fixed `install-zoro.sh` in tag source
- asset `xboard-node-linux-amd64` copied from existing `zoro-v0.1.0` release/build
- asset `SHA256SUMS`

No need to upload `xbctl-linux-amd64` to Zoro release because installer intentionally pulls it from official upstream latest.

## Notes

No production touched. No Horivex touched. No protocol/payment/order/admin bundle changes.
