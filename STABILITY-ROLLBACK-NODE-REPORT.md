# Stability rollback node report

## Date

2026-06-01 ~12:52 GMT+7

## Action

```text
cd /opt/xboard-node-dev
git reset --hard 18bea3a
```

## Removed commits

- `6e0fe2c Add online refresh node report`
- `3f4ebed Improve online report refresh interval`

## Retained commits (VLESS TCP HTTP header fix)

```
18bea3a Add Zoro release report
8a7d5bd Package Zoro install script for VLESS TCP HTTP header fix
6341926 Fix VLESS TCP HTTP header for sing-box
e7efe53 Fix VLESS TCP HTTP header support
```

## Source state after rollback

- `internal/config/config.go`: no `report_interval_seconds` field
- `internal/service/service.go`: no `ReportIntervalSeconds` logic, clamp back to min 5s (default upstream)
- `config.yml.example`: no online refresh docs
- `git status --short`: only untracked `build/` (expected)

## Binary

`build/xboard-node-custom` was from the online refresh build. If a rebuild is needed, use the same Docker build command with the reverted source. Currently zoro-v0.1.0 binary is on node test (confirmed working).

## GitHub

- No push during online refresh, so remote is clean.
- Branch `fix/vless-tcp-http-header` still at `18bea3a`.
- Tag `zoro-v0.1.0` still at `18bea3a`.

## Summary

Node source is back to stable `zoro-v0.1.0`. VLESS TCP HTTP header fix preserved. No unwanted changes remain.
