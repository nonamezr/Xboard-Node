# Xboard Node custom binary test runbook

Scope: test only a non-production node. Do not use production token/API key. Do not run on Horivex.

## 1. Backup current binary/config

```bash
sudo mkdir -p /root/xboard-node-backup-$(date +%Y%m%d-%H%M%S)
BACKUP_DIR=$(ls -td /root/xboard-node-backup-* | head -1)
sudo cp -a /usr/local/bin/xboard-node "$BACKUP_DIR/xboard-node.bak"
sudo cp -a /etc/xboard-node "$BACKUP_DIR/etc-xboard-node.bak"
```

## 2. Stop service

```bash
sudo systemctl stop xboard-node
```

## 3. Install custom binary

Copy `/opt/xboard-node-dev/build/xboard-node-custom` to the test node, then:

```bash
sudo install -m 0755 xboard-node-custom /usr/local/bin/xboard-node
```

## 4. Start service

```bash
sudo systemctl start xboard-node
sudo systemctl status xboard-node --no-pager
```

## 5. Check log

```bash
journalctl -u xboard-node -f
```

Confirm no `unknown version: 71` when testing VLESS TCP HTTP header.

## 6. Client test

Use V2Box/Shadowrocket against a test node config only:

- Protocol: VLESS
- Network: tcp
- Header type: http
- Host: www.softbank.jp
- Path: /
- TLS/security: none

## 7. Rollback if needed

```bash
sudo systemctl stop xboard-node
sudo install -m 0755 "$BACKUP_DIR/xboard-node.bak" /usr/local/bin/xboard-node
sudo cp -a "$BACKUP_DIR/etc-xboard-node.bak" /etc/xboard-node
sudo systemctl start xboard-node
journalctl -u xboard-node -f
```
