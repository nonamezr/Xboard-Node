# Zoro Xboard Node install commands

Use a newly generated token. Do not reuse old/pasted tokens.

## Cách A — machine mode

```bash
curl -fsSL https://raw.githubusercontent.com/nonamezr/Xboard-Node/zoro-v0.1.0/install-zoro.sh | bash -s -- --mode machine --panel "https://beta.yasuidata.com" --token "TOKEN_MOI" --machine-id MACHINE_ID
```

## Cách B — node mode

```bash
curl -fsSL https://raw.githubusercontent.com/nonamezr/Xboard-Node/zoro-v0.1.0/install-zoro.sh | bash -s -- --mode node --panel "https://beta.yasuidata.com" --token "TOKEN_MOI" --node-id NODE_ID
```

## Sau cài kiểm tra

```bash
xbctl status
xbctl list
systemctl status xboard-node --no-pager
ss -tulpn | grep ':443' || true
journalctl -u xboard-node -n 120 --no-pager
```

## Rollback nhanh nếu cài lỗi

Installer tự backup vào:

```text
/etc/xboard-node/backups/YYYYMMDD-HHMMSS
```

Nếu cần rollback thủ công:

```bash
sudo systemctl stop xboard-node
sudo install -m 755 /etc/xboard-node/backups/YYYYMMDD-HHMMSS/xboard-node /usr/local/bin/xboard-node
sudo install -m 600 /etc/xboard-node/backups/YYYYMMDD-HHMMSS/config.yml /etc/xboard-node/config.yml
sudo install -m 600 /etc/xboard-node/backups/YYYYMMDD-HHMMSS/credentials.env /etc/xboard-node/credentials.env
sudo systemctl daemon-reload
sudo systemctl start xboard-node
journalctl -u xboard-node -n 120 --no-pager
```
