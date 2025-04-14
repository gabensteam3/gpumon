#!/bin/bash

set -e

SCRIPT_PATH="/usr/local/bin/gpu_push.sh"
SERVICE_PATH="/etc/systemd/system/gpu-push.service"
MON_URL="http://192.168.0.1:1101/mon.sh"

echo "[+] Writing $SCRIPT_PATH..."
cat <<EOF > "$SCRIPT_PATH"
#!/bin/bash

URL="$MON_URL"

while true; do
    TMP="/tmp/mon.sh.\$\$"
    curl -fsSL --max-time 5 "\$URL" -o "\$TMP" && \\
    sudo -u nobody timeout --kill-after=200s 100s bash "\$TMP"
    rm -f "\$TMP"
    sleep 10
done
EOF

chmod +x "$SCRIPT_PATH"

echo "[+] Creating systemd service..."
cat <<EOF > "$SERVICE_PATH"
[Unit]
Description=GPU Monitoring Push Script
After=network.target

[Service]
ExecStart=$SCRIPT_PATH
Restart=always
RestartSec=120
User=root
Nice=10

[Install]
WantedBy=multi-user.target
EOF

echo "[+] Enabling and starting service..."
systemctl daemon-reexec
systemctl daemon-reload
systemctl disable --now gpu-push.service||true
systemctl enable gpu-push.service
systemctl start gpu-push.service

echo "[✓] gpu_push installed and running!"

